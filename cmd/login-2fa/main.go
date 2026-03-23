package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	qrterminal "github.com/mdp/qrterminal/v3"
	"makeiso/login2fa/internal/login2fa"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "machine-code":
		runMachineCode(os.Args[2:])
	case "generate":
		runGenerate(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  login-2fa machine-code [--json]")
	fmt.Println("  login-2fa generate --machine-code CODE")
	fmt.Println("  login-2fa verify --machine-code CODE --code 123456")
}

func runMachineCode(args []string) {
	fs := flag.NewFlagSet("machine-code", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "print machine details as json")
	_ = fs.Parse(args)

	code, material := login2fa.LocalMachineCode()
	if *jsonOut {
		payload := map[string]any{
			"machine_code": code,
			"material":     material,
		}
		writeJSON(payload)
		return
	}
	fmt.Println("Machine code:", code)
	qrText := terminalQR(code)
	if qrText != "" {
		fmt.Println(qrText)
	}
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	machineCode := fs.String("machine-code", "", "machine code")
	timestamp := fs.Int64("timestamp", login2fa.NowUnix(), "unix timestamp")
	step := fs.Int("step", login2fa.DefaultStep, "time step")
	digits := fs.Int("digits", login2fa.DefaultDigits, "code digits")
	jsonOut := fs.Bool("json", false, "print json output")
	_ = fs.Parse(args)

	if *machineCode == "" {
		exitOnError(fmt.Errorf("--machine-code is required"))
	}
	exitOnError(login2fa.ValidateBuiltins())
	masterKey, err := login2fa.ResolveMasterKey()
	exitOnError(err)
	value, err := login2fa.GenerateCode(masterKey, *machineCode, *timestamp, *step, *digits)
	exitOnError(err)
	displayCode := login2fa.FormatMachineCode(*machineCode)

	if *jsonOut {
		writeJSON(map[string]any{
			"machine_code": displayCode,
			"timestamp":    *timestamp,
			"step":         *step,
			"digits":       *digits,
			"code":         value,
		})
		return
	}
	fmt.Println(value)
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	machineCode := fs.String("machine-code", "", "machine code")
	timestamp := fs.Int64("timestamp", login2fa.NowUnix(), "unix timestamp")
	step := fs.Int("step", login2fa.DefaultStep, "time step")
	digits := fs.Int("digits", login2fa.DefaultDigits, "code digits")
	window := fs.Int("window", login2fa.DefaultWindow, "allowed time window")
	code := fs.String("code", "", "one time code")
	jsonOut := fs.Bool("json", false, "print json output")
	_ = fs.Parse(args)

	if *code == "" {
		exitOnError(fmt.Errorf("--code is required"))
	}
	if *machineCode == "" {
		exitOnError(fmt.Errorf("--machine-code is required"))
	}

	exitOnError(login2fa.ValidateBuiltins())
	masterKey, err := login2fa.ResolveMasterKey()
	exitOnError(err)
	ok, err := login2fa.VerifyCode(*code, masterKey, *machineCode, *timestamp, *step, *digits, *window)
	exitOnError(err)
	displayCode := login2fa.FormatMachineCode(*machineCode)
	if *jsonOut {
		writeJSON(map[string]any{
			"machine_code": displayCode,
			"verified":     ok,
			"timestamp":    *timestamp,
			"window":       *window,
		})
	}
	if !*jsonOut {
		if ok {
			fmt.Println("OK")
		} else {
			fmt.Println("FAIL")
		}
	}
	if !ok {
		os.Exit(1)
	}
}

func writeJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	exitOnError(err)
	fmt.Println(string(data))
}

func exitOnError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(2)
}

func terminalQR(machineCode string) string {
	var buffer bytes.Buffer
	qrterminal.GenerateHalfBlock(machineCode, qrterminal.L, &buffer)
	return strings.TrimSpace(buffer.String())
}
