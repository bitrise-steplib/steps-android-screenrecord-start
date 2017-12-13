package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/bitrise-tools/go-steputils/tools"
)

// ConfigsModel ...
type ConfigsModel struct {
	RecordID       string
	EmulatorSerial string
}

type adbModel struct {
	adbBinPath string
	serial     string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		RecordID:       os.Getenv("record_id"),
		EmulatorSerial: os.Getenv("emulator_serial"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- RecordID: %s", configs.RecordID)
	log.Printf("- EmulatorSerial: %s", configs.EmulatorSerial)
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfNotEmpty(configs.RecordID); err != nil {
		return fmt.Errorf("RecordID, error: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.EmulatorSerial); err != nil {
		return fmt.Errorf("EmulatorSerial, error: %s", err)
	}

	return nil
}

func (model adbModel) shell(commands ...string) (string, error) {
	cmd := command.New(model.adbBinPath, append([]string{"-s", model.serial, "shell"}, commands...)...)
	return cmd.RunAndReturnTrimmedCombinedOutput()
}

func (model adbModel) shellDetached(commands ...string) (string, error) {
	cmd := command.New(model.adbBinPath, append([]string{"-s", model.serial, "shell"}, commands...)...)
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

	adb := adbModel{adbBinPath: adbBinPath, serial: configs.EmulatorSerial}

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
		return fmt.Errorf("screenrecord already running or unable to check if screenrecord is running")
	}

	log.Donef("- Done")
	fmt.Println()

	log.Infof("Start recording")
	_, err = adb.shellDetached(fmt.Sprintf("screenrecord /data/local/tmp/%s.mp4 &", configs.RecordID))
	if err != nil {
		return fmt.Errorf("failed to run adb command, error: %s, output: %s", err, out)
	}
	if err := tools.ExportEnvironmentWithEnvman("BITRISE_RECORD_ID", configs.RecordID); err != nil {
		log.Warnf("Failed to export environment (BITRISE_RECORD_ID), error: %s", err)
	}

	time.Sleep(2 * time.Second)

	log.Printf("- Check if screen recording started")
	out, err = adb.shell("ps | grep screenrecord | cat")
	if err != nil {
		return fmt.Errorf("failed to run adb command, error: %s, output: %s", err, out)
	}

	if out == "" {
		return fmt.Errorf("screenrecord didn't started")
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
