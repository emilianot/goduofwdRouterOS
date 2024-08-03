/*
	Based (copied) on code from the goFwd project
	Credits to jftuga (John Taylor) for the original code.
	https://github.com/jftuga/gofwd
*/

package main

import (
	"fmt"
	"net/url"
	"strings"

	duoapi "github.com/duosecurity/duo_api_golang"
	"github.com/duosecurity/duo_api_golang/authapi"
)

func duoCheck(listenerName string, duoName string, IP string) string {
	var err error

	if len(listenerName) == 0 {
		listenerName = "goDuoFwdRos"
	}

	duoClient := duoapi.NewDuoApi(config.DuoCreds[duoName].Integration, config.DuoCreds[duoName].Secret, config.DuoCreds[duoName].Hostname, "go-client")
	if duoClient == nil {
		fmt.Printf("ERROR: Could not create Duo client for %s\n", duoName)
		return ""
	}
	duoAuthClient := authapi.NewAuthApi(*duoClient)
	check, err := duoAuthClient.Check()
	if err != nil {
		fmt.Printf("DUO ERROR %s: %s\n", duoName, err)
		return ""
	}
	if check == nil {
		fmt.Printf("DUO ERROR %s: check is nil\n", duoName)
		return ""
	}

	if check.StatResult.Stat != "OK" {
		fmt.Printf("DUO ERROR %s: %s %s\n", duoName, *check.StatResult.Message, *check.StatResult.Message_Detail)
		return ""
	}

	options := []func(*url.Values){authapi.AuthUsername(config.DuoCreds[duoName].Username)}
	options = append(options, authapi.AuthDevice("auto"))
	options = append(options, authapi.AuthDisplayUsername("From IP: "+IP))
	options = append(options, authapi.AuthType(listenerName))
	result, err := duoAuthClient.Auth("push", options...)
	if err != nil {
		fmt.Printf("DUO ERROR %s: %s\n", duoName, err)
		return ""
	}
	if result == nil {
		fmt.Printf("DUO ERROR %s: result is nil\n", duoName)
		return ""
	}

	fmt.Printf("DUO %+v\n", result)
	if result.StatResult.Stat == "OK" {
		if result.Response.Result == "allow" {
			fmt.Printf("DUO OK %s %s %s\n", IP, result.StatResult.Stat, result.Response.Result)
			return "allow"
		} else if result.Response.Result == "deny" && !strings.Contains(result.Response.Status_Msg, "timed out") {
			fmt.Printf("DUO DENY %s %s %s\n", IP, result.StatResult.Stat, result.Response.Result)
			return "deny"
		}
	}
	fmt.Printf("DUO DENIED %s %+v\n", IP, result.StatResult)
	return ""
}
