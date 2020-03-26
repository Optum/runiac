package params_test

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.optum.com/healthcarecloud/terrascale/mocks"
	"github.optum.com/healthcarecloud/terrascale/pkg/params"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sut params.StepParameters
var logger *logrus.Entry

func TestMain(m *testing.M) {

	logger = logrus.New().WithField("", "")

	flag.Parse()
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

func TestGetParameter_ShouldRetrieveParametersCorrectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	params.ParametersList = nil

	stubParamStoreResponse := &ssm.GetParametersByPathOutput{
		NextToken: nil,
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/param-testing"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting"),
				Version:          nil,
			},
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/param-testing"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("csptesting"),
				Version:          nil,
			},
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/gibberish/param-testinggibberish"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("gibberish"),
				Version:          nil,
			},
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/track/step/param-testing2"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("csp/stage/track/step/param-testing2"),
				Version:          nil,
			},
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/track/step/ring/param-testing2"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("csp/stage/track/step/ring/param-testing2"),
				Version:          nil,
			},
		},
	}

	stubSSMAPI := mocks.NewMockSSMAPI(ctrl)
	// assert 5 steps are executed
	stubSSMAPI.EXPECT().GetParametersByPath(gomock.Any()).Return(stubParamStoreResponse, nil).MaxTimes(1)

	sut := &params.AWSParamStore{
		Ssmapi: stubSSMAPI,
	}

	mock := sut.GetParamsForStep(logger, "csp", "stage", "track", "step", "ring")

	assert.Equal(t, map[string]string{
		"testing":  "cspstagetesting",
		"testing2": "csp/stage/track/step/ring/param-testing2",
	}, mock)
}

func TestGetParameters_ShouldBeCaseAgnostic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	params.ParametersList = nil

	stubParamStoreResponse := &ssm.GetParametersByPathOutput{
		NextToken: nil,
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/param-testing"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting"),
				Version:          nil,
			},
		},
	}

	stubSSMAPI := mocks.NewMockSSMAPI(ctrl)
	// assert 5 steps are executed
	stubSSMAPI.EXPECT().GetParametersByPath(gomock.Any()).Return(stubParamStoreResponse, nil).MaxTimes(1)

	sut := &params.AWSParamStore{
		Ssmapi: stubSSMAPI,
	}

	mock := sut.GetParamsForStep(logger, "CSP", "stage", "track", "step", "ring")

	assert.Equal(t, map[string]string{
		"testing": "cspstagetesting",
	}, mock)
}

func TestGetParameters_ShouldNotRepeatParamsFetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	params.ParametersList = nil

	stubParamStoreResponse := &ssm.GetParametersByPathOutput{
		NextToken: nil,
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/param-testing"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting"),
				Version:          nil,
			},
		},
	}

	stubSSMAPI := mocks.NewMockSSMAPI(ctrl)
	stubSSMAPI.EXPECT().GetParametersByPath(gomock.Any()).Return(stubParamStoreResponse, nil).MaxTimes(1)

	sut := &params.AWSParamStore{
		Ssmapi: stubSSMAPI,
	}

	mock1 := sut.GetParamsForStep(logger, "CSP", "stage", "track", "step", "ring")
	mock2 := sut.GetParamsForStep(logger, "CSP", "stage", "track", "step", "ring")

	assert.Equal(t, map[string]string{
		"testing": "cspstagetesting",
	}, mock1)
	assert.Equal(t, mock1, mock2)
}

func TestGetParamsForStep_ShouldHandleTruncatedResponsesCorrectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	params.ParametersList = nil

	// Set up responses. All but the last stubbed response return NextToken,
	// which indicates there are more parameters that need to be fetched yet

	stubParamStoreResponse1 := &ssm.GetParametersByPathOutput{
		NextToken: aws.String("next-token1"),
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/param-testing1"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting1"),
				Version:          nil,
			},
		},
	}

	stubParamStoreResponse2 := &ssm.GetParametersByPathOutput{
		NextToken: aws.String("next-token2"),
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/param-testing2"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting2"),
				Version:          nil,
			},
		},
	}

	stubParamStoreResponse3 := &ssm.GetParametersByPathOutput{
		NextToken: aws.String("next-token3"),
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/param-testing3"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting3"),
				Version:          nil,
			},
		},
	}

	// Parameter is the same name as the one in stubParamStoreResponse2,
	// but this is deeper in the namespace so it takes priority
	stubParamStoreResponse4 := &ssm.GetParametersByPathOutput{
		NextToken: nil,
		Parameters: []*ssm.Parameter{
			{
				ARN:              aws.String("stub"),
				LastModifiedDate: nil,
				Name:             aws.String("/bedrock/delivery/csp/stage/track/step/ring/param-testing2"),
				Selector:         nil,
				SourceResult:     nil,
				Type:             nil,
				Value:            aws.String("cspstagetesting4"),
				Version:          nil,
			},
		},
	}

	maxResultsPerQuery := int64(10)
	withDecryption := true
	stubSSMAPI := mocks.NewMockSSMAPI(ctrl)

	// GetParametersByPath should be called 4 times since the first 3 results are truncated
	gomock.InOrder(
		stubSSMAPI.EXPECT().GetParametersByPath(&ssm.GetParametersByPathInput{
			MaxResults:     &maxResultsPerQuery,
			Path:           aws.String("/bedrock/delivery"),
			Recursive:      aws.Bool(true),
			WithDecryption: &withDecryption,
			NextToken:      nil,
		}).Return(stubParamStoreResponse1, nil),
		stubSSMAPI.EXPECT().GetParametersByPath(&ssm.GetParametersByPathInput{
			MaxResults:     &maxResultsPerQuery,
			Path:           aws.String("/bedrock/delivery"),
			Recursive:      aws.Bool(true),
			WithDecryption: &withDecryption,
			NextToken:      aws.String("next-token1"),
		}).Return(stubParamStoreResponse1, nil).Return(stubParamStoreResponse2, nil),
		stubSSMAPI.EXPECT().GetParametersByPath(&ssm.GetParametersByPathInput{
			MaxResults:     &maxResultsPerQuery,
			Path:           aws.String("/bedrock/delivery"),
			Recursive:      aws.Bool(true),
			WithDecryption: &withDecryption,
			NextToken:      aws.String("next-token2"),
		}).Return(stubParamStoreResponse3, nil),
		stubSSMAPI.EXPECT().GetParametersByPath(&ssm.GetParametersByPathInput{
			MaxResults:     &maxResultsPerQuery,
			Path:           aws.String("/bedrock/delivery"),
			Recursive:      aws.Bool(true),
			WithDecryption: &withDecryption,
			NextToken:      aws.String("next-token3"),
		}).Return(stubParamStoreResponse4, nil),
	)

	sut := &params.AWSParamStore{
		Ssmapi: stubSSMAPI,
	}

	mock := sut.GetParamsForStep(logger, "CSP", "stage", "track", "step", "ring")

	assert.Equal(t, map[string]string{
		"testing1": "cspstagetesting1",
		"testing2": "cspstagetesting4", // The deeper parameter is expected here
		"testing3": "cspstagetesting3",
	}, mock)
}

//func TestGetConfig_ShouldCorrectlyGatherParams(t *testing.T) {
//	auth := &auth.SDKAuthenticator{
//		Logger:              logrus.New().WithField("", ""),
//		BedrockCommonRegion: "us-east-1",
//	}
//
//	sut := &params.AWSParamStore{
//		Ssmapi: ssm.New(auth.GetPlatformParametersSession()),
//	}
//
//	mock := sut.GetParamsForStep(logrus.New().WithField("", ""), "csp", "stage", "track", "step", "ring")
//
//	assert.Equal(t, map[string]string{
//		"testing":  "cspstagetesting",
//		"testing2": "csp/stage/track/step/ring/param-testing2",
//	}, mock)
//}
