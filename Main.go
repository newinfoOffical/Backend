/*
File Name:  Main.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	_ "embed" // Required for embedding the default Config file
	"fmt"
	"os"

	"github.com/PeernetOfficial/core"
	"github.com/google/uuid"
)

const configFile = "Config.yaml"
const appName = "Peernet Browser"

//go:embed "Config Default.yaml"
var ConfigDefault []byte

var config struct {
	// Warning: These settings are currently overwritten (deleted) when the config file is updated by core.
	// In the future the core package will consider custom config fields.

	// API settings
	APIListen          []string  `yaml:"APIListen"`          // WebListen is in format IP:Port and declares where the web-interface should listen on. IP can also be ommitted to listen on any.
	APIUseSSL          bool      `yaml:"APIUseSSL"`          // Enables SSL.
	APICertificateFile string    `yaml:"APICertificateFile"` // This is the certificate received from the CA. This can also include the intermediate certificate from the CA.
	APICertificateKey  string    `yaml:"APICertificateKey"`  // This is the private key.
	APITimeoutRead     string    `yaml:"APITimeoutRead"`     // The maximum duration for reading the entire request, including the body.
	APITimeoutWrite    string    `yaml:"APITimeoutWrite"`    // The maximum duration before timing out writes of the response. This includes processing time and is therefore the max time any HTTP function may take.
	APIKey             uuid.UUID `yaml:"APIKey"`             // API key. Empty UUID 00000000-0000-0000-0000-000000000000 = not used.
	DebugAPI           bool      `yaml:"DebugAPI"`           // Enables the debug API which allows profiling. Do not enable in production. Only available if compiled with debug tag.

	// Update
	UpdateFolder string `yaml:"UpdateFolder"` // Folder to temporarily store update files
	PluginFolder string `yaml:"PluginFolder"` // Plugin folder
}

func main() {
	userAgent := appName + "/" + core.Version

	filters := &core.Filters{
		DHTSearchStatus:        filterSearchStatus,
		IncomingRequest:        filterIncomingRequest,
		MessageIn:              filterMessageIn,
		MessageOutAnnouncement: filterMessageOutAnnouncement,
		MessageOutResponse:     filterMessageOutResponse,
		MessageOutTraverse:     filterMessageOutTraverse,
		MessageOutPing:         filterMessageOutPing,
		MessageOutPong:         filterMessageOutPong,
	}

	core.ConfigDefault = ConfigDefault

	backend, status, err := core.Init(userAgent, configFile, filters, &config)

	if status != core.ExitSuccess {
		switch status {
		case core.ExitErrorConfigAccess:
			fmt.Printf("Unknown error accessing config file '%s': %s\n", configFile, err.Error())
		case core.ExitErrorConfigRead:
			fmt.Printf("Error reading config file '%s': %s\n", configFile, err.Error())
		case core.ExitErrorConfigParse:
			fmt.Printf("Error parsing config file '%s' (make sure it is valid YAML format): %s\n", configFile, err.Error())
		case core.ExitErrorLogInit:
			fmt.Printf("Error opening log file '%s': %s\n", backend.Config.LogFile, err.Error())
		default:
			fmt.Printf("Unknown error %d initializing backend: %s\n", status, err.Error())
		}
		os.Exit(status)
	}

	apiListen, apiKey, watchPID := parseCmdParams()

	// Exit if the API is not specified via command-line and the config file. This check must happen after the config was parsed.
	// This is the backend and an API must be provided for a frontend.
	// This behavior makes sure that if the user accidentally starts this executable, it will do nothing.
	if len(apiListen) == 0 && len(config.APIListen) == 0 {
		os.Exit(core.ExitParamWebapiInvalid)
		return
	}

	backend.Stdout.Subscribe(os.Stdout)

	update(backend)

	startAPI(backend, apiListen, apiKey)

	go processExitMonitor(backend, watchPID)

	backend.Connect()

	userCommands(backend, os.Stdin, os.Stdout, nil)
}
