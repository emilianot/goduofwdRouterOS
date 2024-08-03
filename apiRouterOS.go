package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/net-byte/go-gateway"
)

func addIpToAddressList(ip string, list string, timeout int) {
	var payload string = config.Routeros.PostPayload

	fmt.Printf("\nAdding IP %s to list %s (timeout %v)\n", ip, list, timeout)

	if len(ip) == 0 {
		fmt.Printf("IP is empty\n")
		return
	}

	if len(list) == 0 {
		fmt.Printf("list is empty\n")
		return
	}

	if strings.Contains(config.Routeros.Url, "$GATEWAY") {
		gatewayIP, err := gateway.DiscoverGatewayIPv4()
		if err != nil {
			fmt.Println("Gateway IP: error:", err)
			return
		}
		fmt.Printf("Gateway IP: %s\n", gatewayIP.String())
		config.Routeros.Url = strings.ReplaceAll(config.Routeros.Url, "$GATEWAY", gatewayIP.String())
	}

	payload = strings.ReplaceAll(payload, "$IP", ip)
	payload = strings.ReplaceAll(payload, "$LIST", list)
	payload = strings.ReplaceAll(payload, "$TIMEOUT", fmt.Sprintf("%d", timeout))

	fmt.Printf("RouterOS URL: %s\n", config.Routeros.Url)
	fmt.Printf("Payload: %s\n", payload)

	r, err := http.NewRequest("PUT", config.Routeros.Url, strings.NewReader(payload))
	if err != nil {
		fmt.Printf("error creating request: %s", err.Error())
		return
	}
	r.SetBasicAuth(config.Routeros.Username, config.Routeros.Password)
	r.Header.Add("Content-Type", "application/json")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Routeros.InsecureSkipVerify},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(r)
	if err != nil {
		fmt.Printf("error adding IP to list: %s", err.Error())
		return
	}
	//defer resp.Body.Close()
	if resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 400 {
		fmt.Printf("IP %s added to list %s (timeout %v)\n", ip, list, timeout)
	} else {
		fmt.Printf("[ERROR] adding IP %s to list %v (timeout %v). REST API ERROR: %s", ip, list, timeout, resp.Status)
	}
}
