//go:build response && !noresponse

package response

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func promptCreds() (login, password string, err error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", "", fmt.Errorf("need a terminal to type the login")
	}
	defer tty.Close()
	fmt.Fprint(tty, "login: ")
	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return "", "", err
	}
	login = strings.TrimSpace(line)
	if login == "" || strings.ContainsAny(login, "\r\n\x00") {
		return "", "", fmt.Errorf("login is empty")
	}
	fmt.Fprint(tty, "password: ")
	pw, err := term.ReadPassword(int(tty.Fd()))
	fmt.Fprintln(tty)
	if err != nil {
		return "", "", err
	}
	password = string(pw)
	for i := range pw {
		pw[i] = 0
	}
	if password == "" || strings.ContainsAny(password, "\r\n\x00") {
		return "", "", fmt.Errorf("password is empty")
	}
	return login, password, nil
}
