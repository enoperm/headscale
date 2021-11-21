package headscale

import (
	"gopkg.in/check.v1"
	"inet.af/netaddr"
)

func (s *Suite) TestGetAvailableIp(c *check.C) {
	ips, err := h.getAvailableIPs()

	c.Assert(err, check.IsNil)

	expected := netaddr.MustParseIP("10.27.0.1")

	c.Assert(len(ips), check.Equals, 1)
	c.Assert(ips[0].String(), check.Equals, expected.String())
}

func (s *Suite) TestGetUsedIps(c *check.C) {
	ips, err := h.getAvailableIPs()
	c.Assert(err, check.IsNil)

	n, err := h.CreateNamespace("test_ip")
	c.Assert(err, check.IsNil)

	pak, err := h.CreatePreAuthKey(n.Name, false, false, nil)
	c.Assert(err, check.IsNil)

	_, err = h.GetMachine("test", "testmachine")
	c.Assert(err, check.NotNil)

	m := Machine{
		ID:             0,
		MachineKey:     "foo",
		NodeKey:        "bar",
		DiscoKey:       "faa",
		Name:           "testmachine",
		NamespaceID:    n.ID,
		Registered:     true,
		RegisterMethod: "authKey",
		AuthKeyID:      uint(pak.ID),
		IPAddresses:    ips.ToStringSlice(),
	}
	h.db.Save(&m)

	usedIps, err := h.getUsedIPs()

	c.Assert(err, check.IsNil)

	expected := netaddr.MustParseIP("10.27.0.1")

	c.Assert(len(usedIps), check.Equals, 1)
	c.Assert(usedIps[0], check.Equals, expected)

	m1, err := h.GetMachineByID(0)
	c.Assert(err, check.IsNil)

	c.Assert(len(m1.IPAddresses), check.Equals, 1)
	c.Assert(m1.IPAddresses[0], check.Equals, expected.String())
}

func (s *Suite) TestGetMultiIp(c *check.C) {
	n, err := h.CreateNamespace("test-ip-multi")
	c.Assert(err, check.IsNil)

	for i := 1; i <= 350; i++ {
		ips, err := h.getAvailableIPs()
		c.Assert(err, check.IsNil)

		pak, err := h.CreatePreAuthKey(n.Name, false, false, nil)
		c.Assert(err, check.IsNil)

		_, err = h.GetMachine("test", "testmachine")
		c.Assert(err, check.NotNil)

		m := Machine{
			ID:             uint64(i),
			MachineKey:     "foo",
			NodeKey:        "bar",
			DiscoKey:       "faa",
			Name:           "testmachine",
			NamespaceID:    n.ID,
			Registered:     true,
			RegisterMethod: "authKey",
			AuthKeyID:      uint(pak.ID),
			IPAddresses:    ips.ToStringSlice(),
		}
		h.db.Save(&m)
	}

	usedIps, err := h.getUsedIPs()

	c.Assert(err, check.IsNil)

	c.Assert(len(usedIps), check.Equals, 350)

	c.Assert(usedIps[0], check.Equals, netaddr.MustParseIP("10.27.0.1"))
	c.Assert(usedIps[9], check.Equals, netaddr.MustParseIP("10.27.0.10"))
	c.Assert(usedIps[300], check.Equals, netaddr.MustParseIP("10.27.1.45"))

	// Check that we can read back the IPs
	m1, err := h.GetMachineByID(1)
	c.Assert(err, check.IsNil)
	c.Assert(len(m1.IPAddresses), check.Equals, 1)
	c.Assert(
		m1.IPAddresses[0],
		check.Equals,
		netaddr.MustParseIP("10.27.0.1").String(),
	)

	m50, err := h.GetMachineByID(50)
	c.Assert(err, check.IsNil)
	c.Assert(len(m50.IPAddresses), check.Equals, 1)
	c.Assert(
		m50.IPAddresses[0],
		check.Equals,
		netaddr.MustParseIP("10.27.0.50").String(),
	)

	expectedNextIP := netaddr.MustParseIP("10.27.1.95")
	nextIP, err := h.getAvailableIPs()
	c.Assert(err, check.IsNil)

	c.Assert(len(nextIP), check.Equals, 1)
	c.Assert(nextIP[0].String(), check.Equals, expectedNextIP.String())

	// If we call get Available again, we should receive
	// the same IP, as it has not been reserved.
	nextIP2, err := h.getAvailableIPs()
	c.Assert(err, check.IsNil)

	c.Assert(len(nextIP2), check.Equals, 1)
	c.Assert(nextIP2[0].String(), check.Equals, expectedNextIP.String())
}

func (s *Suite) TestGetAvailableIpMachineWithoutIP(c *check.C) {
	ips, err := h.getAvailableIPs()
	c.Assert(err, check.IsNil)

	expected := netaddr.MustParseIP("10.27.0.1")

	c.Assert(len(ips), check.Equals, 1)
	c.Assert(ips[0].String(), check.Equals, expected.String())

	n, err := h.CreateNamespace("test_ip")
	c.Assert(err, check.IsNil)

	pak, err := h.CreatePreAuthKey(n.Name, false, false, nil)
	c.Assert(err, check.IsNil)

	_, err = h.GetMachine("test", "testmachine")
	c.Assert(err, check.NotNil)

	m := Machine{
		ID:             0,
		MachineKey:     "foo",
		NodeKey:        "bar",
		DiscoKey:       "faa",
		Name:           "testmachine",
		NamespaceID:    n.ID,
		Registered:     true,
		RegisterMethod: "authKey",
		AuthKeyID:      uint(pak.ID),
	}
	h.db.Save(&m)

	ips2, err := h.getAvailableIPs()
	c.Assert(err, check.IsNil)

	c.Assert(len(ips2), check.Equals, 1)
	c.Assert(ips2[0].String(), check.Equals, expected.String())
}
