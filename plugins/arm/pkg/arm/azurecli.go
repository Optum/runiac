package arm

type AzureRM interface {
	ResourceDelete(options *Options, ids []string) (out string, err error)
	SubCreate(options *Options, deploymentName string, accountID string, location string, file string) (out string, err error)
	SubDelete(options *Options, deploymentName string, accountID string) (out string, err error)
	SubShow(options *Options, deploymentName string, accountID string) (out string, err error)
	SubWhatIf(options *Options, deploymentName string, accountID string, location string, file string) (out string, err error)
	Version(options *Options) (out string, err error)
}

type AzureCLI struct{}

func (a AzureCLI) ResourceDelete(options *Options, ids []string) (out string, err error) {
	return ResourceDelete(options, ids)
}

func (a AzureCLI) SubCreate(options *Options, deploymentName string, accountID string, location string, file string) (out string, err error) {
	return SubCreate(options, deploymentName, accountID, location, file)
}

func (a AzureCLI) SubShow(options *Options, deploymentName string, accountID string) (out string, err error) {
	return SubShow(options, deploymentName, accountID)
}

func (a AzureCLI) SubDelete(options *Options, deploymentName string, accountID string) (out string, err error) {
	return SubDelete(options, deploymentName, accountID)
}

func (a AzureCLI) SubWhatIf(options *Options, deploymentName string, accountID string, location string, file string) (out string, err error) {
	return SubWhatIf(options, deploymentName, accountID, location, file)
}

func (a AzureCLI) Version(options *Options) (out string, err error) {
	return Version(options)
}
