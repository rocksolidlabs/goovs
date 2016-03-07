package goovs

func (client *ovsClient) PortExists(brname, portname string) (bool, error) {
    return false, nil
}

func (client *ovsClient) CreateInternalPort(brname, portname string) error {
    return nil
}
 
func (client *ovsClient) CreateVethPort(brname, portname string) error {
    return nil
}