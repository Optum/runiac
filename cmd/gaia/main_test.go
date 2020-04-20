package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var DefaultStubAccountID = "1"
var StubVersion = "v0.0.5"

func TestMain(m *testing.M) {
	os.Setenv("ACCOUNT_ID", DefaultStubAccountID)
	os.Setenv("CSP", "AWS")
	os.Setenv("CREDS_ID", "1")
	os.Setenv("CODEPIPELINE_EXECUTION_ID", "1")
	os.Setenv("STEP_FUNCTION_NAME", "1")
	os.Setenv("UPDATE_STATUS_LAMBDA", "1")

	os.Setenv("ENVIRONMENT", "local")
	os.Setenv("NAMESPACE", "1")

	fs = afero.NewMemMapFs()
	log = logrus.New().WithField("environment", "unittest")

	var stubVersionJSON = []byte(fmt.Sprintf(`{
	  "version": "%s",
	  "pr_region": "",
	  "base_image": "common-bedrock-customer-deploy:v0.0.3"
	}`, StubVersion))

	afero.WriteFile(fs, "version.json", stubVersionJSON, 0644)

	flag.Parse()
	exitCode := m.Run()

	_ = os.Remove("testbackend.tf")

	// Exit
	os.Exit(exitCode)
}
