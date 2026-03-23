package main

/*
#cgo LDFLAGS: -lpam
#include <security/pam_appl.h>
#include <security/pam_ext.h>
#include <stdlib.h>

static int prompt_hidden(pam_handle_t *pamh, const char *msg, char **resp_out) {
	const struct pam_conv *conv = NULL;
	struct pam_message message;
	const struct pam_message *message_ptr = &message;
	struct pam_response *responses = NULL;
	const void *item = NULL;
	int retval = pam_get_item(pamh, PAM_CONV, &item);
	if (retval != PAM_SUCCESS || item == NULL) {
		return PAM_SYSTEM_ERR;
	}
	conv = (const struct pam_conv *)item;
	if (conv == NULL || conv->conv == NULL) {
		return PAM_SYSTEM_ERR;
	}
	message.msg_style = PAM_PROMPT_ECHO_OFF;
	message.msg = msg;
	retval = conv->conv(1, &message_ptr, &responses, conv->appdata_ptr);
	if (retval != PAM_SUCCESS) {
		return retval;
	}
	if (responses == NULL || responses[0].resp == NULL) {
		return PAM_AUTH_ERR;
	}
	*resp_out = responses[0].resp;
	return PAM_SUCCESS;
}

*/
import "C"

import (
	"bytes"
	"os"
	"strings"
	"time"
	"unsafe"

	qrterminal "github.com/mdp/qrterminal/v3"
	"makeiso/login2fa/internal/login2fa"
)

func main() {}

//export pam_sm_setcred
func pam_sm_setcred(pamh *C.pam_handle_t, flags C.int, argc C.int, argv **C.char) C.int {
	_ = pamh
	_ = flags
	_ = argc
	_ = argv
	return C.PAM_SUCCESS
}

//export pam_sm_authenticate
func pam_sm_authenticate(pamh *C.pam_handle_t, flags C.int, argc C.int, argv **C.char) C.int {
	_ = flags
	_ = argc
	_ = argv
	if err := login2fa.ValidateBuiltins(); err != nil {
		return C.PAM_AUTHINFO_UNAVAIL
	}
	masterKey, err := login2fa.ResolveMasterKey()
	if err != nil {
		return C.PAM_AUTHINFO_UNAVAIL
	}
	machineCode, _ := login2fa.LocalMachineCode()
	qrText := terminalQR(machineCode)

	var response *C.char
	promptText := "Machine code: " + machineCode
	if qrText != "" {
		promptText += "\n" + qrText
	}
	promptText += "\nOne-time verification code: "
	prompt := C.CString(promptText)
	defer C.free(unsafe.Pointer(prompt))
	if retval := C.prompt_hidden(pamh, prompt, &response); retval != C.PAM_SUCCESS {
		return retval
	}
	if response == nil {
		return C.PAM_AUTH_ERR
	}
	code := C.GoString(response)
	C.free(unsafe.Pointer(response))

	verified, err := login2fa.VerifyCode(
		code,
		masterKey,
		machineCode,
		time.Now().Unix(),
		login2fa.DefaultStep,
		login2fa.DefaultDigits,
		login2fa.DefaultWindow,
	)
	if err != nil {
		return C.PAM_AUTH_ERR
	}
	if !verified {
		return C.PAM_AUTH_ERR
	}

	_ = os.Setenv("LOGIN_2FA_MACHINE_CODE", machineCode)
	return C.PAM_SUCCESS
}

func terminalQR(machineCode string) string {
	var buffer bytes.Buffer
	qrterminal.GenerateHalfBlock(machineCode, qrterminal.L, &buffer)
	return strings.TrimSpace(buffer.String())
}
