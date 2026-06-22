package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Znevna/zteONU/app/factory"
	"github.com/Znevna/zteONU/app/telnet"
	"github.com/Znevna/zteONU/version"
)

var (
	// Used for flags.
	user           string
	passwd         string
	ip             string
	port           int
	permTelnet     bool
	telnetPort     int
	SecLvl         int
	zadmin         bool
	disableV6      bool
	reboot         bool
	userList       []string
	passwdList     []string
	defaultUsers   = []string{"factorymode", "telecomadmin", "admin", "CMCCAdmin", "CUAdmin", "cqadmin", "user", "admin", "cuadmin", "lnadmin", "useradmin"}
	defaultPasswds = []string{"nE%jA@5b", "nE7jA%5m", "admin", "aDm8H%MdA", "CUAdmin", "cqunicom", "1620@CTCC", "1620@CUcc", "admintelecom", "cuadmin", "lnadmin"}

	rootCmd = &cobra.Command{
		Use: "zteOnu",
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(); err != nil {
				fmt.Println(err)
			}
		},
	}
	showVersion    bool
)

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "Factory mode auth username (If not provided, a known list will be used)")
	rootCmd.PersistentFlags().StringVarP(&passwd, "pass", "p", "", "Factory mode auth password (If not provided, a known list will be used)")
	rootCmd.PersistentFlags().StringVarP(&ip, "ip", "i", "192.168.1.1", "ONU ip address")
	rootCmd.PersistentFlags().IntVar(&port, "port", 80, "ONU http port")
	rootCmd.PersistentFlags().BoolVar(&permTelnet, "telnet", false, "Enable permanent telnet (port 2323, user: tadmin, pass: dynamically generated)")
	rootCmd.PersistentFlags().IntVar(&SecLvl, "seclvl", 3, "Security level for telnet access, if you got \"Access Denied\", try 2 or 1.\nUse with --telnet flag")
	rootCmd.PersistentFlags().IntVar(&telnetPort, "tp", 23, "ONU telnet port")
	rootCmd.PersistentFlags().BoolVar(&zadmin, "zadmin", false, "Add zadmin user (password: dynamically generated)")
	rootCmd.PersistentFlags().BoolVar(&disableV6, "disable-lan-v6", false, "Disable LAN RA/DHCPv6 services and block them on LAN1")
	rootCmd.PersistentFlags().BoolVar(&reboot, "reboot", false, "Reboot the ONU after applying changes")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Print current version of zteOnu")
}

func run() error {
	version.Show()
	if showVersion {
		return nil
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	magicMac, err := net.ParseMAC("00:07:29:55:35:57")
	if err != nil {
		return err
	}

	var isMagicMac bool
	for _, i := range interfaces {
		if i.HardwareAddr != nil && bytes.Equal(i.HardwareAddr, magicMac) {
			isMagicMac = true
			break
		}
	}

	if !isMagicMac {
		return errors.New(`
======================================================================
 [ ERROR: LOCAL MAC ADDRESS VERIFICATION FAILED ]

 You MUST spoof your NIC MAC address to:  00:07:29:55:35:57

 Please apply the spoofed MAC and run the tool again.
======================================================================`)
	}

	// User default lists if user\pass not passed
	if user == "" {
		userList = defaultUsers
	} else {
		userList = []string{user}
	}
	if passwd == "" {
		passwdList = defaultPasswds
	} else {
		passwdList = []string{passwd}
	}
	// Check list size
	if len(userList) != len(passwdList) {
		return errors.New("Users and Passwords list should have same length")
	}

	var tlUser string
	var tlPass string

	success := false
	for i := 0; i < len(userList); i++ {

		var err error
		for count := 1; count <= 5; count++ {

			tlUser, tlPass, err = factory.New(userList[i], passwdList[i], ip, port).Handle()
			if err != nil {
				errMsg := err.Error()
				if idx := strings.Index(errMsg, "connectex:"); idx != -1 {
					errMsg = errMsg[idx:]
				} else if idx := strings.Index(errMsg, "dial tcp"); idx != -1 {
					errMsg = errMsg[idx:]
				}
				fmt.Printf("\n  -> [%d/5] Payload delivery failed (%s). Retrying...\n", count, errMsg)
				time.Sleep(time.Millisecond * 500)
				continue
			}

			fmt.Printf("Successfully authenticated with user: %s and password: %s\n", userList[i], passwdList[i])
			success = true
			break
		}

		if success {
			break
		}
	}
	if tlUser != "" && tlPass != "" {
		fmt.Println(strings.Repeat("-", 35))
		fmt.Printf("Telnet Credentials (!! Temporary !!)\nUser: %s\nPass: %s\n", tlUser, tlPass)
	}

	if permTelnet || zadmin || disableV6 || reboot {
		// create telnet conn
		t, err := telnet.New(tlUser, tlPass, ip, telnetPort)
		if err != nil {
			return err
		}
		defer t.Close()

		if permTelnet {
			fmt.Println(strings.Repeat("-", 35))
			// handle permanent telnet
			if err := t.PermTelnet(SecLvl, tlPass); err != nil {
				return err
			} else {
				fmt.Printf("Permanent Telnet succeeded\r\nUser: tadmin\nPass: %s\n", tlPass)
			}
		}

		if zadmin {
			fmt.Println(strings.Repeat("-", 35))
			if err := t.AddSuperAdmin(tlPass); err != nil {
				return err
			} else {
				fmt.Printf("Added zadmin user successfully\r\nUser: zadmin\nPass: %s\n", tlPass)
			}
		}

		if disableV6 {
			fmt.Println(strings.Repeat("-", 35))
			if err := t.DisableV6(); err != nil {
				return err
			} else {
				fmt.Println("Disabled RA Service and DHCPv6 Server on LAN successfully")
			}
		}

		if reboot {
			// reboot device
			fmt.Println(strings.Repeat("-", 35))
			fmt.Println("Rebooting device...")
			if err := t.Reboot(); err != nil {
				return err
			}
		} else {
			if permTelnet || zadmin || disableV6 {
				fmt.Println(strings.Repeat("-", 35))
				fmt.Println("Changes applied successfully!")
			}
		}
	}
	return nil
}

func Execute() error {
	return rootCmd.Execute()
}
