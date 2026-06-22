package telnet

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func New(user string, pass string, ip string, port int) (*Telnet, error) {
	var conn net.Conn
	var err error

	for i := 0; i < 5; i++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
		if err == nil {
			break
		}
		errMsg := strings.TrimPrefix(err.Error(), fmt.Sprintf("dial tcp %s:%d: ", ip, port))
		fmt.Printf("[%d/5] Telnet connection failed (%s). Retrying...\n", i+1, errMsg)
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to telnet after retries: %v", err)
	}

	t := &Telnet{
		user:    user,
		pass:    pass,
		conn:    conn,
		timeout: 5 * time.Second,
	}

	if err := t.loginTelnet(); err != nil {
		t.Close()
		return nil, fmt.Errorf("telnet login failed: %v", err)
	}

	return t, nil
}

func (t *Telnet) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

func (t *Telnet) PermTelnet(SecLvl int, tempPass string) error {
	if err := t.modifyDB(SecLvl, tempPass); err != nil {
		return err
	}

	return nil
}


func (t *Telnet) loginTelnet() error {
	// Wait for login prompt
	t.expect("Login: ", "Username: ")
	t.conn.Write([]byte(t.user + crlf))

	t.expect("Password: ")
	t.conn.Write([]byte(t.pass + crlf))

	_, _, err := t.expect("# ", "> ")
	if err != nil {
		return err
	}

	// Ping the shell with an empty line to ensure it is fully initialized and ready
	t.conn.Write([]byte(crlf))
	t.expect("# ", "> ")

	return nil
}

func (t *Telnet) modifyDB(SecLvl int, tempPass string) error {
	fmt.Println("Configuring TelnetCfg (Row 0)...")
	// Request the current TelnetCfg table to detect supported fields
	cmdStr := "sendcmd 1 DB p TelnetCfg"
	cmd := []byte(cmdStr + crlf)
	if _, err := t.conn.Write(cmd); err != nil {
		return err
	}

	out, _, err := t.expect("# ", "> ")
	if err != nil {
		return fmt.Errorf("failed to get TelnetCfg: %v", err)
	}

	// Extract all <DM name="..."> fields to understand what the router supports
	supportedFields := make(map[string]bool)
	re := regexp.MustCompile(`<DM name="([^"]+)"`)
	matches := re.FindAllStringSubmatch(out, -1)
	for _, match := range matches {
		if len(match) > 1 {
			supportedFields[match[1]] = true
		}
	}

	// Build the dynamic payload
	fmt.Println("  -> Updating TelnetCfg to enable LAN/TS and set user 'tadmin'")
	prefix := "sendcmd 1 DB set TelnetCfg 0 "
	var commands []string

	if supportedFields["TS_Enable"] {
		commands = append(commands, prefix+"TS_Enable 1 > /dev/null")
	}
	if supportedFields["Lan_Enable"] {
		commands = append(commands, prefix+"Lan_Enable 1 > /dev/null")
	}
	if supportedFields["TSLan_UName"] {
		commands = append(commands, prefix+"TSLan_UName tadmin > /dev/null")
	}
	if supportedFields["TSLan_UPwd"] {
		commands = append(commands, prefix+"TSLan_UPwd "+tempPass+" > /dev/null")
	}
	if supportedFields["TS_UName"] {
		commands = append(commands, prefix+"TS_UName tadmin > /dev/null")
	}
	if supportedFields["TS_UPwd"] {
		commands = append(commands, prefix+"TS_UPwd "+tempPass+" > /dev/null")
	}
	if supportedFields["TS_Port"] {
		commands = append(commands, prefix+"TS_Port 2323 > /dev/null")
	}
	if supportedFields["TSLan_Port"] {
		commands = append(commands, prefix+"TSLan_Port 2323 > /dev/null")
	}
	if supportedFields["InitSecLvl"] {
		commands = append(commands, prefix+"InitSecLvl "+strconv.Itoa(SecLvl)+" > /dev/null")
	}
	if supportedFields["Max_Con_Num"] {
		commands = append(commands, prefix+"Max_Con_Num 3 > /dev/null")
	}

	// Save changes
	commands = append(commands, "sendcmd 1 DB save")

	if err := t.sendCmd(commands...); err != nil {
		return err
	}

	return nil
}

func (t *Telnet) DisableV6() error {
	fmt.Println("Configuring IPv6 Settings...")
	commands := []string{
		"sendcmd 1 DB set DHCP6SPool 0 Enable 0 > /dev/null",
		"sendcmd 1 DB set RAIS 0 Enable 0 > /dev/null",
		"sendcmd 1 DB save",
	}

	if err := t.sendCmd(commands...); err != nil {
		return err
	}

	return nil
}

func (t *Telnet) expect(prompts ...string) (string, string, error) {
	t.conn.SetReadDeadline(time.Now().Add(t.timeout))
	defer t.conn.SetReadDeadline(time.Time{})
	buf := make([]byte, 1024)
	var output string
	for {
		n, err := t.conn.Read(buf)
		if err != nil {
			return output, "", err
		}

		output += string(buf[:n])
		for _, p := range prompts {
			if strings.HasSuffix(output, p) {
				return output, p, nil
			}
		}
	}
}

func (t *Telnet) sendCmd(commands ...string) error {
	for _, command := range commands {
		cmd := []byte(command + crlf)

		n, err := t.conn.Write(cmd)
		if err != nil {
			return fmt.Errorf("failed to send command %s: %v", command, err)
		}

		if expected, actual := len(cmd), n; expected != actual {
			return fmt.Errorf("transmission problem: tried sending %d bytes, but actually only sent %d bytes for command %s", expected, actual, command)
		}

		// Wait for the prompt to confirm the command finished executing
		if _, _, err := t.expect("# ", "> "); err != nil {
			return fmt.Errorf("timeout waiting for prompt after command %s: %v", command, err)
		}
	}

	return nil
}

func (t *Telnet) Reboot() error {
	if err := t.sendCmd("reboot"); err != nil {
		return err
	}

	return nil
}

func (t *Telnet) AddSuperAdmin(tempPass string) error {
	cmdStr := "sendcmd 1 DB p DevAuthInfo"
	cmd := []byte(cmdStr + crlf)
	if _, err := t.conn.Write(cmd); err != nil {
		return err
	}

	out, _, err := t.expect("# ", "> ")
	if err != nil {
		return fmt.Errorf("failed to get DevAuthInfo: %v", err)
	}

	re := regexp.MustCompile(`RowCount="(\d+)"`)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 2 {
		return fmt.Errorf("could not find RowCount in response")
	}

	rowCount, err := strconv.Atoi(matches[1])
	if err != nil || rowCount == 0 {
		return fmt.Errorf("invalid RowCount %s", matches[1])
	}

	fmt.Printf("Parsed DevAuthInfo XML: Found %d active credential rows\n", rowCount)
	fmt.Println("Searching for existing 'zadmin' alias...")

	// Parse the XML dump to find the existing zadmin alias
	var existingIdx = -1
	rows := strings.Split(out, `<Row No="`)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		idxEnd := strings.Index(row, `"`)
		if idxEnd == -1 {
			continue
		}

		if strings.Contains(row, `name="Alias" val="zadmin"`) {
			idxStr := row[:idxEnd]
			parsedIdx, err := strconv.Atoi(idxStr)
			if err == nil {
				existingIdx = parsedIdx
				break
			}
		}
	}

	var commands []string
	if existingIdx != -1 {
		// Found existing zadmin! Only update the password.
		fmt.Printf("  -> Found 'zadmin' at index %d! Updating password only...\n", existingIdx)
		prefix := fmt.Sprintf("sendcmd 1 DB set DevAuthInfo %d ", existingIdx)
		commands = []string{
			prefix + "Pass " + tempPass + " > /dev/null",
			"sendcmd 1 DB save",
		}
	} else {
		// We must explicitly add a new row because zadmin was not found
		// The new row will be created at index = rowCount
		idx := rowCount
		fmt.Printf("  -> 'zadmin' alias not found. Injecting new root user at index %d...\n", idx)
		prefix := fmt.Sprintf("sendcmd 1 DB set DevAuthInfo %d ", idx)
		commands = []string{
			"sendcmd 1 DB addr DevAuthInfo > /dev/null",
			prefix + fmt.Sprintf("ViewName IGD.AU%d > /dev/null", idx+1),
			prefix + "Alias zadmin > /dev/null",
			prefix + "User zadmin > /dev/null",
			prefix + "Pass " + tempPass + " > /dev/null",
			prefix + "AppID 1 > /dev/null",
			prefix + "Level 1 > /dev/null",
			prefix + "Enable 1 > /dev/null",
			"sendcmd 1 DB save",
		}
	}

	if err := t.sendCmd(commands...); err != nil {
		return err
	}

	return nil
}
