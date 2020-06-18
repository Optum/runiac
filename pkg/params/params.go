//go:generate mockgen -destination=../../mocks/mock_ssmapi.go -package=mocks github.com/aws/aws-sdk-go/service/ssm/ssmiface SSMAPI

package params

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/sirupsen/logrus"
)

type StepParameters interface {
	GetParamsForStep(logger *logrus.Entry, csp string, stage string, track string, step string, ring string) (params map[string]string)
}

type AWSParamStore struct {
	Ssmapi ssmiface.SSMAPI
}

var ParametersList []*ssm.Parameter

func (p AWSParamStore) GetParamsForStep(logger *logrus.Entry, csp string, stage string, track string, step string, ring string) (params map[string]string) {

	params = map[string]string{}

	paramNamespace := "/bedrock/delivery"
	withDecryption := true
	var nextToken *string

	var input *ssm.GetParametersByPathInput
	var maxResultsPerQuery int64 = 10 // Max is 10

	// Grab all parameters from SSM under the namespace.
	// Handles truncated responses since GetParametersByPath
	// only returns 10 (max) parameters per request
	if len(ParametersList) == 0 {
		for {
			input = &ssm.GetParametersByPathInput{
				MaxResults:     aws.Int64(maxResultsPerQuery),
				Path:           aws.String(paramNamespace),
				Recursive:      aws.Bool(true),
				WithDecryption: &withDecryption,
				NextToken:      nextToken,
			}

			paramResponse, err := p.Ssmapi.GetParametersByPath(input)
			nextToken = paramResponse.NextToken

			if err != nil {
				logger.WithError(err).Error("GetParametersByPath failure")
				return
			}

			if len(paramResponse.Parameters) < 1 {
				logger.Error("No parameters retrieved...")
				return
			}

			logger.Debugf("Retrieved %v params", len(paramResponse.Parameters))
			ParametersList = append(ParametersList, paramResponse.Parameters...)

			if nextToken == nil {
				break // NextToken was nil, there are no more parameters to retrieve
			}
		}
	}

	// hierarchy = /bedrock/delivery/{csp}/{stage}/{track}/{step}/{ring}/param-{parameter}

	depthCache := make(map[string]int)
	depthMatch := strings.ToLower(csp + stage + track + step + ring)

	for i := 0; i < len(ParametersList); i += 1 {
		var name string

		cleanedParamName := strings.Replace(*ParametersList[i].Name, "/bedrock/delivery/", "", -1)

		// check for bedrock param name from aws parameter name
		split := strings.Split(cleanedParamName, "/")

		rawParam := split[len(split)-1]

		heirarchyNamespaces := split[:len(split)-1]

		// capture param depth to compare priority of parameter
		paramDepth := len(heirarchyNamespaces)

		// join namespaces to compare to incoming targets
		joinedNamespaces := strings.ToLower(strings.Join(heirarchyNamespaces, ""))

		// short circuit if csp, stage, track, step, or ring does not match
		if !strings.HasPrefix(depthMatch, joinedNamespaces) {
			logger.Debugf("depthMatch did not match for %s (%s:%s)", rawParam, depthMatch, joinedNamespaces)
			continue
		}

		if strings.HasPrefix(rawParam, "param-") {
			name = strings.Split(rawParam, "param-")[1]
		}

		// short circuit if param name could not be derived
		if name == "" {
			continue
		}

		// short circuit if parameter has already been set at a higher depth
		if val, ok := depthCache[name]; ok {
			if paramDepth <= val {
				continue
			}
		}

		value := *ParametersList[i].Value
		params[name] = value
		depthCache[name] = paramDepth
	}
	logger.Debugf("Matched %v params", len(params))
	return
}
