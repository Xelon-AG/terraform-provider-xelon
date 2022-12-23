package device

// Network represents a helper struct for expand/flatten network conversion.
type Network struct {
	IPAddressID      int
	NetworkID        int
	NICControllerKey int
	NICKey           int
	NICNumber        int
	NICUnit          int
}

func ExpandNetwork(l []interface{}) *Network {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	n := l[0].(map[string]interface{})

	network := &Network{
		IPAddressID:      n["ipv4_address_id"].(int),
		NetworkID:        n["id"].(int),
		NICControllerKey: n["nic_controller_key"].(int),
		NICKey:           n["nic_key"].(int),
		NICNumber:        n["nic_number"].(int),
		NICUnit:          n["nic_unit"].(int),
	}

	return network
}
