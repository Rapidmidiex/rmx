package provider

type ProviderCfg struct {
	BaseURI                string
	ClientID, ClientSecret string
	HashKey                []byte
	EncKey                 []byte
}
