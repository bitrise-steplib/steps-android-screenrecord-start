package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-tools/go-steputils/input"

	"github.com/bitrise-io/depman/pathutil"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// ConfigsModel ...
type ConfigsModel struct {
	ID             string
	EmulatorSerial string
}

type adb struct {
	adbBinPath string
	serial     string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		ID:             os.Getenv("id"),
		EmulatorSerial: os.Getenv("emulator_serial"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- ID: %s", configs.ID)
	log.Printf("- EmulatorSerial: %s", configs.EmulatorSerial)
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfNotEmpty(configs.ID); err != nil {
		return fmt.Errorf("ID, error: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.EmulatorSerial); err != nil {
		return fmt.Errorf("EmulatorSerial, error: %s", err)
	}

	return nil
}

func (props adb) shell(commands ...string) (string, error) {
	cmd := command.New(props.adbBinPath, append([]string{"-s", props.serial, "shell"}, commands...)...)
	return cmd.RunAndReturnTrimmedCombinedOutput()
}

func (props adb) shellDetached(commands ...string) (string, error) {
	cmd := command.New(props.adbBinPath, append([]string{"-s", props.serial, "shell"}, commands...)...)
	rCmd := cmd.GetCmd()
	var b bytes.Buffer
	rCmd.Stdout = &b
	rCmd.Stderr = &b
	err := rCmd.Start()
	return b.String(), err
}

func mainE() error {
	// Input validation
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		log.Errorf("Issue with input: %s", err)
		os.Exit(1)
	}

	fmt.Println()

	//
	// Main
	log.Infof("Checking compability")
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		return fmt.Errorf("no ANDROID_HOME set")
	}
	adbBinPath := filepath.Join(androidHome, "platform-tools/adb")
	exists, err := pathutil.IsPathExists(adbBinPath)
	if err != nil {
		return fmt.Errorf("failed to check if path exists: %s, error: %s", adbBinPath, err)
	}
	if !exists {
		return fmt.Errorf("adb binary doesn't exist at: %s", adbBinPath)
	}

	adb := adb{adbBinPath: adbBinPath, serial: configs.EmulatorSerial}

	out, err := adb.shell("which screenrecord")
	if err != nil {
		return fmt.Errorf("failed to run adb command, error: %s, output: %s", err, out)
	}
	if out == "" {
		return fmt.Errorf("screenrecord binary is not available on the device")
	}
	out, err = adb.shell("ps | grep screenrecord | cat")
	if err != nil {
		return fmt.Errorf("failed to run adb command, error: %s, output: %s", err, out)
	}
	if out != "" {
		return fmt.Errorf("screenrecord already running")
	}
	log.Donef("- Done")
	fmt.Println()

	log.Infof("Start recording")
	_, err = adb.shellDetached(fmt.Sprintf("screenrecord /data/local/tmp/%s.mp4 &", configs.ID))
	if err != nil {
		return fmt.Errorf("failed to run adb command, error: %s, output: %s", err, out)
	}
	log.Donef("- Started")

	return nil
}

func main() {
	err := mainE()
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}
}
