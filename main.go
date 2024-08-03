package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

type TGeo struct {
	Country  string  `yaml:"country"`
	Region   string  `yaml:"region"`
	City     string  `yaml:"city"`
	Postal   string  `yaml:"postal"`
	Org      string  `yaml:"org"`
	Hostname string  `yaml:"hostname"`
	Lon      float64 `yaml:"lon"`
	Lat      float64 `yaml:"lat"`
	Distance float64 `yaml:"distance"`
}

type TConfig struct {
	Routeros struct {
		Url                string `yaml:"url"`
		Username           string `yaml:"username"`
		Password           string `yaml:"password"`
		PostPayload        string `yaml:"postPayload"`
		InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	} `yaml:"routeros"`
	IPInfo struct {
		Token         string `yaml:"token"`
		CacheTime     int64  `yaml:"cacheTime"`
		CacheMaxCount int    `yaml:"cacheMaxCount"`
	} `yaml:"ipinfo"`
	DuoCreds map[string]struct {
		Username    string `yaml:"username"`
		Secret      string `yaml:"secret"`
		Integration string `yaml:"integration"`
		Hostname    string `yaml:"hostname"`
	} `yaml:"duoCreds"`
	Listeners []struct {
		Name             string `yaml:"name"`
		Proto            string `yaml:"proto"`
		Port             int32  `yaml:"port"`
		DuoName          string `yaml:"duoName"`
		AllowListName    string `yaml:"allowListName"`
		AllowListTimeout int    `yaml:"allowListTimeout"`
		DenyListName     string `yaml:"denyListName"`
		DenyListTimeout  int    `yaml:"denyListTimeout"`
		Geos             []TGeo `yaml:"geos"`
	} `yaml:"listeners"`
}

var config TConfig
var meLon float64 = 0.0
var meLat float64 = 0.0
var checkingIP = make(map[string]bool)

func main() {
	var cfgFile string = "config.yaml"

	if len(os.Args) > 1 && len(os.Args[1]) > 5 && strings.HasSuffix(strings.ToLower(os.Args[1]), ".yaml") {
		cfgFile = os.Args[1]
	}

	fmt.Println("Config File: ", cfgFile)
	yamlFile, err := os.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}
	fmt.Println("Readed config")

	fmt.Println("Routeros: ")
	fmt.Println("\tURL: ", config.Routeros.Url)
	fmt.Println("\tInsecureSkipVerify: ", config.Routeros.InsecureSkipVerify)
	fmt.Println("\tUsername: ", config.Routeros.Username)
	fmt.Println("\tPassword: ", config.Routeros.Password)
	fmt.Println("\tPostPayload: ", config.Routeros.PostPayload)

	fmt.Println("\nIP Info: ")
	fmt.Println("\tToken: ", config.IPInfo.Token)
	fmt.Println("\tCache Time: ", config.IPInfo.CacheTime)
	fmt.Println("\tCache Max Count: ", config.IPInfo.CacheMaxCount)

	fmt.Println("\n\nDuoCreds: ")
	for k, v := range config.DuoCreds {
		if len(v.Username) == 0 || len(v.Secret) == 0 || len(v.Integration) == 0 || len(v.Hostname) == 0 {
			fmt.Printf("\n\t - INVALID: %+v\n", v)
			panic("Invalid DuoCreds")
		}
		fmt.Println("\t - Name: ", k)
		fmt.Println("\t   Username: ", v.Username)
		fmt.Println("\t   Secret: ", v.Secret)
		fmt.Println("\t   Integration: ", v.Integration)
		fmt.Println("\t   Hostname: ", v.Hostname)
	}

	fmt.Println("\n\nListeners: ")
	for lk, v := range config.Listeners {
		v.Proto = strings.ToUpper(v.Proto)
		config.Listeners[lk].Proto = v.Proto

		if v.Proto != "TCP" && v.Proto != "UDP" {
			fmt.Printf("\n\t - INVALID: %+v\n", v)
			panic("Invalid Listener PROTO")
		}
		if v.Port <= 0 {
			fmt.Printf("\n\t - INVALID: %+v\n", v)
			panic("Invalid Listener PORT")
		}
		if len(v.AllowListName) == 0 {
			fmt.Printf("\n\t - INVALID: %+v\n", v)
			panic("Invalid Listener ListName")
		}
		if len(v.Geos) == 0 && len(v.DuoName) == 0 {
			fmt.Printf("\n\t - INVALID: %+v\n", v)
			panic("Invalid Listener No Rules")
		}
		if len(v.DuoName) > 0 && config.DuoCreds[v.DuoName].Username == "" {
			fmt.Printf("\n\t - INVALID: %+v\n", v)
			panic("Invalid Listener DuoName")
		}

		fmt.Println("\t - Name: ", v.Name)
		fmt.Println("\t   Proto: ", v.Proto)
		fmt.Println("\t   Port: ", v.Port)
		fmt.Println("\t   DuoName: ", v.DuoName)
		fmt.Println("\t   Allow ListName: ", v.AllowListName, " Timeout: ", v.AllowListTimeout)
		fmt.Println("\t   Deny ListName: ", v.DenyListName, " Timeout: ", v.DenyListTimeout)
		fmt.Println("\t   Geos: ")
		for _, g := range v.Geos {
			if !(len(g.Country) > 0 || len(g.Region) > 0 || len(g.City) > 0 || len(g.Postal) > 0 || len(g.Org) > 0 || len(g.Hostname) > 0 || g.Distance > 0) {
				fmt.Printf("\t\t - INVALID: %+v\n", g)
				panic("Invalid Geo rule")
			}
			fmt.Println("\t\t - Country: ", iif(len(g.Country) > 0, g.Country, "*ALL*"))
			fmt.Println("\t\t   Region: ", iif(len(g.Region) > 0, g.Region, "*ALL*"))
			fmt.Println("\t\t   City: ", iif(len(g.City) > 0, g.City, "*ALL*"))
			fmt.Println("\t\t   Postal: ", iif(len(g.Postal) > 0, g.Postal, "*ALL*"))
			fmt.Println("\t\t   Org: ", iif(len(g.Org) > 0, g.Org, "*ALL*"))
			fmt.Println("\t\t   Hostname: ", iif(len(g.Hostname) > 0, g.Hostname, "*ALL*"))
			if g.Distance > 0 {
				fmt.Println("\t\t   Distance: ", g.Distance, " Lat: ", g.Lat, " Lon: ", g.Lon)
			} else {
				fmt.Println("\t\t   Distance: *ALL*")
			}
		}
	}

	fmt.Println("\n\nStarting listeners")
	for k, _ := range config.Listeners {
		if strings.ToUpper(config.Listeners[k].Proto) == "TCP" {
			go startListenerTCP(k)
		} else {
			go startListenerUDP(k)
		}
	}

	// Wait for a SIGINT or SIGTERM signal to gracefully shut down the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

func startListenerTCP(k int) {
	var err error
	fmt.Println("\n\tTCP Port: ", config.Listeners[k].Port)
	server, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(config.Listeners[k].Port)})
	if err != nil {
		fmt.Print("Error listening:", err.Error(), "\n")
		os.Exit(1)
	}
	defer server.Close()
	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		fromIP, fromPort, err := net.SplitHostPort(conn.RemoteAddr().String())
		conn.Close()
		if err != nil {
			fmt.Println("Error parsing IP: ", err.Error())
		} else {
			fmt.Printf("\nConnection from %v:%v to Listener %v (TCP/%v)\n", fromIP, fromPort, config.Listeners[k].Name, config.Listeners[k].Port)
			go checkAccess(k, fromIP)
		}
	}
}

func startListenerUDP(k int) {
	fmt.Print("\n\tUDP Port: ", config.Listeners[k].Port)
	server, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(config.Listeners[k].Port)})
	if err != nil {
		fmt.Print("Error listening:", err.Error(), "\n")
		os.Exit(1)
	}
	defer server.Close()
	for {
		buffer := make([]byte, 1)
		_, addr, err := server.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP: ", err.Error())
		} else {
			fromIP, fromPort, err := net.SplitHostPort(addr.String())
			if err != nil {
				fmt.Println("Error parsing IP: ", err.Error())
			} else {
				fmt.Printf("\nConnection from %v:%v to Listener %v (UDP/%v)\n", fromIP, fromPort, config.Listeners[k].Name, config.Listeners[k].Port)
				go checkAccess(k, fromIP)
			}
		}
	}
}

func checkAccess(k int, fromIP string) {
	if checkingIP[fromIP] {
		fmt.Printf("\nIP %s is already being checked. Skip.\n", fromIP)
		return
	}
	checkingIP[fromIP] = true
	listName, timeOut := passRules(k, fromIP)
	if len(listName) > 0 {
		addIpToAddressList(fromIP, listName, timeOut)
	} else {
		fmt.Printf("\nIP %s is not allowed\n", fromIP)
	}
	delete(checkingIP, fromIP)
}

func passRules(k int, fromIP string) (string, int) {
	fmt.Println("\nChecking IP rules for IP ", fromIP)

	if len(fromIP) == 0 {
		fmt.Println("[DENY] IP is empty")
		return "", 0
	}

	// Check geos
	if !IPisInGeoList(fromIP, config.Listeners[k].Geos) {
		fmt.Printf("\n[DENY] IP %s is not in geo list\n", fromIP)
		return config.Listeners[k].DenyListName, config.Listeners[k].DenyListTimeout
	}

	res := duoCheck(config.Listeners[k].Name, config.Listeners[k].DuoName, fromIP)
	if res == "allow" {
		return config.Listeners[k].AllowListName, config.Listeners[k].AllowListTimeout
	} else if res == "deny" {
		return config.Listeners[k].DenyListName, config.Listeners[k].DenyListTimeout
	}

	return "", 0

}

func iif(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}
