//go:generate mockgen -destination ../../mocks/mock_auth.go -package=mocks github.optum.com/healthcarecloud/terrascale/pkg/auth Authenticator

package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sirupsen/logrus"
)

// Authenticator is the interface for authenticating
type Authenticator interface {
	GetPlatformSession() *session.Session
	GetCredentialEnvVarsForAccount(logger *logrus.Entry, csp string, accountID string, credsID string) (creds map[string]string, err error)
	GetPlatformParametersSession(logger *logrus.Entry) (*session.Session, error)
	GetAWSMasterCreds(logger *logrus.Entry, csp string, credsID string) (*credentials.Credentials, error)
}

// SDKAuthenticator implements the Authenticator interface
type SDKAuthenticator struct {
	Logger              *logrus.Entry
	BedrockCommonRegion string
	credentials         map[string]*credentials.Credentials // by accountID
	masterCredentials   map[string]*credentials.Credentials // by csp-credsID
	platformSession     *session.Session
	sessions            map[string]*session.Session
	AzuCredCache        map[string]*AZUCredentials
}

// AWSCredentials is a struct that represents AWS credentials
type AWSCredentials struct {
	Platform *aws.Config
}

// AZUCredentials is a struct that represents Azure credentials.
// These credentials are the management group Service Principal that
// has access to all the subscriptions in the group. This assumes
// the subscription is already in the Management Group
type AZUCredentials struct {
	ID     string
	Secret string
	Tenant string
}

// GetCredentialEnvVarsForAccount retrieves credentials required for the deployment in a map format
func (cli *SDKAuthenticator) GetCredentialEnvVarsForAccount(logger *logrus.Entry, csp string, accountID string, credsID string) (creds map[string]string, err error) {
	if strings.ToLower(csp) == "aws" {
		var awsCreds credentials.Value
		masterCreds, err := cli.GetAWSMasterCreds(logger, csp, credsID)

		if err != nil {
			logger.WithError(err).Error("failed for getMasterCredsForCredsIDAndCSP")
			return nil, err
		}

		awsCreds, err = cli.getDeploymentAccountAssumeRoleCreds(masterCreds, accountID).Get()

		if err != nil {
			logger.WithError(err).Error("failed for getDeploymentAccountAssumeRoleCreds")
			return nil, err
		}

		creds = map[string]string{
			"AWS_ACCESS_KEY_ID":     awsCreds.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY": awsCreds.SecretAccessKey,
			"AWS_SESSION_TOKEN":     awsCreds.SessionToken,
		}
	} else if strings.ToLower(csp) == "azu" {
		var azuCreds *AZUCredentials
		var azuCredsErr error

		if creds, exists := cli.AzuCredCache[accountID]; exists {
			azuCreds = creds
		} else {
			azuCreds, azuCredsErr = cli.getAZUCreds(logger, credsID)
			if azuCredsErr != nil {
				logger.WithError(azuCredsErr).Error("failed for getAZUCreds")
				return nil, azuCredsErr
			}
			cli.AzuCredCache[accountID] = azuCreds
		}

		creds = map[string]string{
			"ARM_CLIENT_ID":       azuCreds.ID,
			"ARM_CLIENT_SECRET":   azuCreds.Secret,
			"ARM_TENANT_ID":       azuCreds.Tenant,
			"ARM_SUBSCRIPTION_ID": accountID,
		}
	} else {
		return nil, fmt.Errorf("invalid csp provided, %s, expecting: aws or azu", csp)
	}

	return creds, err
}

// GetPlatformSession retrieves a session for the executing Fargate role
func (cli *SDKAuthenticator) GetPlatformSession() *session.Session {
	cli.Logger.Debug("GetPlatformSession()...")

	// this is aws configuration set with fargate role and appropriate region
	if cli.platformSession == nil {
		cli.platformSession = session.Must(session.NewSession(aws.NewConfig().WithRegion(cli.BedrockCommonRegion)))
	}

	return cli.platformSession
}

// GetPlatformParametersSession retrieves a session for the executing Fargate role
func (cli *SDKAuthenticator) GetPlatformParametersSession(logger *logrus.Entry) (*session.Session, error) {
	logger.Debug("Executing GetPlatformParametersSession...")

	// this is aws configuration set with fargate role and appropriate region
	platformSession := cli.GetPlatformSession()

	stsSvc := sts.New(platformSession)

	result, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err != nil {
		logger.WithError(err).Error("failed GetCallerIdentity")
		return nil, err
	}

	// TODO: make this account id configurable to centralize parameter store usage across PRs and Prod?
	assumeRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/BedrockDeployParamStoreAccess", *result.Account)

	// include region in cache key otherwise concurrency errors?
	key := fmt.Sprintf("%v", assumeRoleARN)

	// check for cached config
	if cli.sessions != nil && cli.sessions[key] != nil {
		return cli.sessions[key], nil
	}

	// new creds
	creds := stscreds.NewCredentials(platformSession, assumeRoleARN, func(p *stscreds.AssumeRoleProvider) {
		p.Duration = time.Duration(45) * time.Minute
	})

	checkCreds, err := creds.Get()

	if err != nil {
		logger.WithError(err).Errorf("failed to receive credentials for assume role: %s", assumeRoleARN)
		return nil, err
	}

	if checkCreds.AccessKeyID == "" {
		return nil, errors.New("failed to receive credentials for assume role")
	}

	paramSession := session.Must(session.NewSession(platformSession.Config, aws.NewConfig().WithCredentials(creds)))

	if cli.sessions == nil {
		cli.sessions = map[string]*session.Session{}
	}

	cli.sessions[key] = paramSession

	// assumerole into customer account from creds retrieved above
	return paramSession, nil
}

func (cli *SDKAuthenticator) GetAWSMasterCreds(logger *logrus.Entry, csp string, credsID string) (*credentials.Credentials, error) {

	if csp == "" {
		return nil, errors.New("csp parameter is required")
	}

	if credsID == "" {
		return nil, errors.New("credsID parameter is required")
	}

	if cli.masterCredentials == nil {
		cli.masterCredentials = map[string]*credentials.Credentials{}
	}

	masterCredsKey := fmt.Sprintf("%s-%s", credsID, csp)

	if creds, ok := cli.masterCredentials[masterCredsKey]; ok {
		return creds, nil
	}

	paramNamespace := fmt.Sprintf("/launchpad/delivery/access/%s/%s", strings.ToLower(csp), strings.ToLower(credsID))

	id := fmt.Sprintf("%s/id", paramNamespace)
	secret := fmt.Sprintf("%s/secret", paramNamespace)

	logger.Debugf("Getting parameters from %s", id)
	logger.Debugf("Getting parameters from %s", secret)

	ssmsvc := ssm.New(cli.GetPlatformSession())

	withDecryption := true
	paramResponse, err := ssmsvc.GetParameters(&ssm.GetParametersInput{
		Names: []*string{
			aws.String(id),
			aws.String(secret),
		},
		WithDecryption: &withDecryption,
	})

	if err != nil {
		logger.WithError(err).Error("Unable to authenticate. GetParameters failure")
		return nil, err
	}

	// convert to map for retrieval
	params := make(map[string]string)

	if len(paramResponse.Parameters) < 1 {
		logger.Error("Unable to authenticate. No parameters retrieved...")
		return nil, nil
	}
	for i := 0; i < len(paramResponse.Parameters); i++ {

		name := *paramResponse.Parameters[i].Name
		value := *paramResponse.Parameters[i].Value
		params[name] = value
	}

	staticSession, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(params[id], params[secret], ""),
	})

	// NOTE: 8 hour timeout for master session creds (ie. deployment can no longer authenticate after time expires without explicitly recreating session)
	parentSessionCreds, err := sts.New(staticSession).GetSessionToken(&sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(28800), // 8 hours
	})

	if err != nil {
		logger.WithError(err).Error("Unable to authenticate. Error during sts.New(staticSession).GetSessionToken()")
		return nil, err
	}

	cli.masterCredentials[masterCredsKey] = credentials.NewStaticCredentials(*parentSessionCreds.Credentials.AccessKeyId,
		*parentSessionCreds.Credentials.SecretAccessKey, *parentSessionCreds.Credentials.SessionToken)

	return cli.masterCredentials[masterCredsKey], nil
}

func (cli *SDKAuthenticator) getAZUCreds(logger *logrus.Entry, credsID string) (*AZUCredentials, error) {
	if credsID == "" {
		return nil, errors.New("credsID parameter is required")
	}

	paramNamespace := fmt.Sprintf("/launchpad/delivery/access/%s/%s", strings.ToLower("azu"), strings.ToLower(credsID))
	id := fmt.Sprintf("%s/id", paramNamespace)
	secret := fmt.Sprintf("%s/secret", paramNamespace)
	tenant := fmt.Sprintf("%s/tenant", paramNamespace)

	ssmsvc := ssm.New(cli.GetPlatformSession())
	withDecryption := true
	paramResponse, err := ssmsvc.GetParameters(&ssm.GetParametersInput{
		Names: []*string{
			aws.String(id),
			aws.String(secret),
			aws.String(tenant),
		},
		WithDecryption: &withDecryption,
	})

	if err != nil {
		logger.WithError(err).Error("Unable to authenticate. GetParameters failure")
		return nil, err
	}

	if len(paramResponse.Parameters) < 1 {
		logger.Error("Unable to authenticate. No parameters retrieved...")
		return nil, errors.New("No parameters retrieved")
	}

	params := make(map[string]string)

	for i := 0; i < len(paramResponse.Parameters); i++ {
		name := *paramResponse.Parameters[i].Name
		value := *paramResponse.Parameters[i].Value
		params[name] = value
	}

	return &AZUCredentials{
		ID:     params[id],
		Secret: params[secret],
		Tenant: params[tenant],
	}, nil
}

func (cli *SDKAuthenticator) getDeploymentAccountAssumeRoleCreds(masterCredentials *credentials.Credentials, accountID string) *credentials.Credentials {

	assumeRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", accountID)

	// include region in cache key otherwise concurrency errors?
	key := fmt.Sprintf("%v", assumeRoleARN)

	// check for cached config
	if cli.credentials != nil && cli.credentials[key] != nil {
		return cli.credentials[key]
	}

	s := session.Must(session.NewSession(aws.NewConfig().WithCredentials(masterCredentials)))

	// new creds
	creds := stscreds.NewCredentials(s, assumeRoleARN, func(p *stscreds.AssumeRoleProvider) {
		p.Duration = time.Duration(1) * time.Hour
	})

	if cli.credentials == nil {
		cli.credentials = map[string]*credentials.Credentials{}
	}

	cli.credentials[key] = creds

	// assumerole into customer account from creds retrieved above
	return creds
}
