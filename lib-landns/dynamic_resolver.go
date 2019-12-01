package landns

type DynamicResolver interface {
	Resolver

	UpdateAddresses(AddressesConfig) error
	GetAddresses() (AddressesConfig, error)

	UpdateCnames(CnamesConfig) error
	GetCnames() (CnamesConfig, error)

	UpdateTexts(TextsConfig) error
	GetTexts() (TextsConfig, error)

	UpdateServices(ServicesConfig) error
	GetServices() (ServicesConfig, error)
}
