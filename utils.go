// Codehere is mostly taken from github.com/tailscale/tailscale
// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package headscale

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/nacl/box"
	"inet.af/netaddr"
	"tailscale.com/tailcfg"
	"tailscale.com/types/wgkey"
)

// Error is used to compare errors as per https://dave.cheney.net/2016/04/07/constant-errors
type Error string

func (e Error) Error() string { return string(e) }

func decode(msg []byte, v interface{}, pubKey *wgkey.Key, privKey *wgkey.Private) error {
	return decodeMsg(msg, v, pubKey, privKey)
}

func decodeMsg(msg []byte, v interface{}, pubKey *wgkey.Key, privKey *wgkey.Private) error {
	decrypted, err := decryptMsg(msg, pubKey, privKey)
	if err != nil {
		return err
	}
	// fmt.Println(string(decrypted))
	if err := json.Unmarshal(decrypted, v); err != nil {
		return fmt.Errorf("response: %v", err)
	}
	return nil
}

func decryptMsg(msg []byte, pubKey *wgkey.Key, privKey *wgkey.Private) ([]byte, error) {
	var nonce [24]byte
	if len(msg) < len(nonce)+1 {
		return nil, fmt.Errorf("response missing nonce, len=%d", len(msg))
	}
	copy(nonce[:], msg)
	msg = msg[len(nonce):]

	pub, pri := (*[32]byte)(pubKey), (*[32]byte)(privKey)
	decrypted, ok := box.Open(nil, msg, &nonce, pub, pri)
	if !ok {
		return nil, fmt.Errorf("cannot decrypt response")
	}
	return decrypted, nil
}

func encode(v interface{}, pubKey *wgkey.Key, privKey *wgkey.Private) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return encodeMsg(b, pubKey, privKey)
}

func encodeMsg(b []byte, pubKey *wgkey.Key, privKey *wgkey.Private) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		panic(err)
	}
	pub, pri := (*[32]byte)(pubKey), (*[32]byte)(privKey)
	msg := box.Seal(nonce[:], b, &nonce, pub, pri)
	return msg, nil
}

func (h *Headscale) getAvailableIPs() (ips MachineAddresses, err error) {
	ipPrefixes := h.cfg.IPPrefixes
	for _, ipPrefix := range ipPrefixes {
		var ip *netaddr.IP
		ip, err = h.getAvailableIP(ipPrefix)
		if err != nil {
			return
		}
		ips = append(ips, *ip)
	}
	return
}

// TODO: Is this concurrency safe?
// What would happen if multiple hosts were to register at the same time?
// Would we attempt to assign the same addresses to multiple nodes?
func (h *Headscale) getAvailableIP(ipPrefix netaddr.IPPrefix) (*netaddr.IP, error) {
	usedIps, err := h.getUsedIPs()
	if err != nil {
		return nil, err
	}

	ipPrefixNetworkAddress, ipPrefixBroadcastAddress := func() (netaddr.IP, netaddr.IP) {
		ipRange := ipPrefix.Range()
		return ipRange.From(), ipRange.To()
	}()

	// Get the first IP in our prefix
	ip := ipPrefixNetworkAddress.Next()

	for {
		if !ipPrefix.Contains(ip) {
			return nil, fmt.Errorf("could not find any suitable IP in %s", ipPrefix)
		}

		switch {
		case ip.Compare(ipPrefixBroadcastAddress) == 0:
			fallthrough
		case containsIPs(usedIps, ip):
			fallthrough
		case ip.IsZero() || ip.IsLoopback():
			ip = ip.Next()
			continue

		default:
			return &ip, nil
		}
	}
}

func (h *Headscale) getUsedIPs() ([]netaddr.IP, error) {
	// FIXME: This really deserves a better data model,
	// but this was quick to get running and it should be enough
	// to begin experimenting with a dual stack tailnet.
	var addressesSlices []string
	h.db.Model(&Machine{}).Pluck("ip_addresses", &addressesSlices)

	addresses := make([]string, len(h.cfg.IPPrefixes)*len(addressesSlices))
	for _, slice := range addressesSlices {
		var a AddressStringSlice
		err := a.Scan(slice)
		if err != nil {
			return nil, fmt.Errorf("failed to read ip from database: %w", err)
		}
		addresses = append(addresses, a...)
	}

	ips := make([]netaddr.IP, 0, len(addresses))
	for _, addr := range addresses {
		if addr != "" {
			ip, err := netaddr.ParseIP(addr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse ip from database, %w", err)
			}

			ips = append(ips, ip)
		}
	}

	return ips, nil
}

func containsIPs(ips []netaddr.IP, ip netaddr.IP) bool {
	for _, v := range ips {
		if v == ip {
			return true
		}
	}

	return false
}

func tailNodesToString(nodes []*tailcfg.Node) string {
	temp := make([]string, len(nodes))

	for index, node := range nodes {
		temp[index] = node.Name
	}

	return fmt.Sprintf("[ %s ](%d)", strings.Join(temp, ", "), len(temp))
}

func tailMapResponseToString(resp tailcfg.MapResponse) string {
	return fmt.Sprintf("{ Node: %s, Peers: %s }", resp.Node.Name, tailNodesToString(resp.Peers))
}
