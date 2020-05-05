package command

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	//"bytes"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"time"

	"github.com/AlecAivazis/survey/v2"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

var validFormats = []string{
	"bash",
	"c",
	"csharp",
	"dw",
	"dword",
	"hex",
	"java",
	"js_be",
	"js_le",
	"num",
	"perl",
	"pl",
	"powershell",
	"ps1",
	"py",
	"python",
	"raw",
	"rb",
	"ruby",
	"sh",
	"vbapplication",
	"vbscript",
}

func generate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	config := parseCompileFlags(ctx)
	if config == nil {
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	compile(config, save, rpc)
}

func regenerate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if len(ctx.Args) < 1 {
		fmt.Printf(Warn+"Invalid implant name, see `help %s`\n", consts.RegenerateStr)
		return
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}

	regenerate, err := rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
		ImplantName: ctx.Args[0],
	})
	if err != nil {
		fmt.Printf(Warn+"Failed to regenerate implant %s\n", err)
		return
	}
	if regenerate.File == nil {
		fmt.Printf(Warn + "Failed to regenerate implant (no data)\n")
		return
	}
	saveTo, err := saveLocation(save, regenerate.File.Name)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	err = ioutil.WriteFile(saveTo, regenerate.File.Data, 0500)
	if err != nil {
		fmt.Printf(Warn+"Failed to write to %s\n", err)
		return
	}
	fmt.Printf(Info+"Implant binary saved to: %s\n", saveTo)
}

func saveLocation(save, defaultName string) (string, error) {
	var saveTo string
	if save == "" {
		save, _ = os.Getwd()
	}
	fi, err := os.Stat(save)
	if os.IsNotExist(err) {
		log.Printf("%s does not exist\n", save)
		if strings.HasSuffix(save, "/") {
			log.Printf("%s is dir\n", save)
			os.MkdirAll(save, 0700)
			saveTo, _ = filepath.Abs(path.Join(saveTo, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			saveDir := filepath.Dir(save)
			_, err := os.Stat(saveTo)
			if os.IsNotExist(err) {
				os.MkdirAll(saveDir, 0700)
			}
			saveTo, _ = filepath.Abs(save)
		}
	} else {
		log.Printf("%s does exist\n", save)
		if fi.IsDir() {
			log.Printf("%s is dir\n", save)
			saveTo, _ = filepath.Abs(path.Join(save, defaultName))
		} else {
			log.Printf("%s is not dir\n", save)
			prompt := &survey.Confirm{Message: "Overwrite existing file?"}
			var confirm bool
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return "", errors.New("File already exists")
			}
			saveTo, _ = filepath.Abs(save)
		}
	}
	return saveTo, nil
}

// func generateEgg(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	outFmt := ctx.Flags.String("output-format")
// 	validFmt := false
// 	for _, f := range validFormats {
// 		if f == outFmt {
// 			validFmt = true
// 			break
// 		}
// 	}
// 	if !validFmt {
// 		fmt.Printf(Warn+"Invalid output format: %s", outFmt)
// 		return
// 	}
// 	stagingURL := ctx.Flags.String("listener-url")
// 	if stagingURL == "" {
// 		return
// 	}
// 	save := ctx.Flags.String("save")
// 	config := parseCompileFlags(ctx)
// 	if config == nil {
// 		return
// 	}
// 	config.Format = clientpb.SliverConfig_SHELLCODE
// 	config.IsSharedLib = true
// 	// Find job type (tcp / http)
// 	u, err := url.Parse(stagingURL)
// 	if err != nil {
// 		fmt.Printf(Warn + "listener-url format not supported")
// 		return
// 	}
// 	port, err := strconv.Atoi(u.Port())
// 	if err != nil {
// 		fmt.Printf(Warn+"Invalid port number: %s", err.Error())
// 		return
// 	}
// 	eggConfig := &clientpb.EggConfig{
// 		Host:   u.Hostname(),
// 		Port:   uint32(port),
// 		Arch:   config.GOARCH,
// 		Format: outFmt,
// 	}
// 	switch u.Scheme {
// 	case "tcp":
// 		eggConfig.Protocol = clientpb.EggConfig_TCP
// 	case "http":
// 		eggConfig.Protocol = clientpb.EggConfig_HTTP
// 	case "https":
// 		eggConfig.Protocol = clientpb.EggConfig_HTTPS
// 	default:
// 		eggConfig.Protocol = clientpb.EggConfig_TCP
// 	}
// 	ctrl := make(chan bool)
// 	go spin.Until("Creating stager shellcode...", ctrl)
// 	data, _ := proto.Marshal(&clientpb.EggReq{
// 		EConfig: eggConfig,
// 		Config:  config,
// 	})
// 	resp := <-rpc(&sliverpb.Envelope{
// 		Type: clientpb.MsgEggReq,
// 		Data: data,
// 	}, 45*time.Minute)
// 	ctrl <- true
// 	if resp.Err != "" {
// 		fmt.Printf(Warn+"%s", resp.Err)
// 		return
// 	}
// 	eggResp := &clientpb.Egg{}
// 	err = proto.Unmarshal(resp.Data, eggResp)
// 	if err != nil {
// 		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
// 		return
// 	}
// 	// Don't display raw shellcode out stdout
// 	if save != "" || outFmt == "raw" {
// 		// Save it to disk
// 		saveTo, _ := filepath.Abs(save)
// 		fi, err := os.Stat(saveTo)
// 		if err != nil {
// 			fmt.Printf(Warn+"Failed to generate sliver egg %v\n", err)
// 			return
// 		}
// 		if fi.IsDir() {
// 			saveTo = filepath.Join(saveTo, eggResp.Filename)
// 		}
// 		err = ioutil.WriteFile(saveTo, eggResp.Data, 0700)
// 		if err != nil {
// 			fmt.Printf(Warn+"Failed to write to: %s\n", saveTo)
// 			return
// 		}
// 		fmt.Printf(Info+"Sliver egg saved to: %s\n", saveTo)
// 	} else {
// 		// Display shellcode to stdout
// 		fmt.Println("\n" + Info + "Here's your Egg:")
// 		fmt.Println(string(eggResp.Data))
// 	}
// 	fmt.Printf("\n"+Info+"Successfully started job #%d\n", eggResp.JobID)
// }

// Shared function that extracts the compile flags from the grumble context
func parseCompileFlags(ctx *grumble.Context) *clientpb.ImplantConfig {
	targetOS := strings.ToLower(ctx.Flags.String("os"))
	arch := strings.ToLower(ctx.Flags.String("arch"))

	c2s := []*clientpb.ImplantC2{}

	mtlsC2 := parseMTLSc2(ctx.Flags.String("mtls"))
	c2s = append(c2s, mtlsC2...)

	httpC2 := parseHTTPc2(ctx.Flags.String("http"))
	c2s = append(c2s, httpC2...)

	dnsC2 := parseDNSc2(ctx.Flags.String("dns"))
	c2s = append(c2s, dnsC2...)

	var symbolObfuscation bool
	if ctx.Flags.Bool("debug") {
		symbolObfuscation = false
	} else {
		symbolObfuscation = !ctx.Flags.Bool("skip-symbols")
	}

	if len(mtlsC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 {
		fmt.Printf(Warn + "Must specify at least one of --mtls, --http, or --dns\n")
		return nil
	}

	rawCanaries := ctx.Flags.String("canary")
	canaryDomains := []string{}
	if 0 < len(rawCanaries) {
		for _, canaryDomain := range strings.Split(rawCanaries, ",") {
			if !strings.HasSuffix(canaryDomain, ".") {
				canaryDomain += "." // Ensure we have the FQDN
			}
			canaryDomains = append(canaryDomains, canaryDomain)
		}
	}

	reconnectInterval := ctx.Flags.Int("reconnect")
	maxConnectionErrors := ctx.Flags.Int("max-errors")

	limitDomainJoined := ctx.Flags.Bool("limit-domainjoined")
	limitHostname := ctx.Flags.String("limit-hostname")
	limitUsername := ctx.Flags.String("limit-username")
	limitDatetime := ctx.Flags.String("limit-datetime")

	isSharedLib := false

	format := ctx.Flags.String("format")
	var configFormat clientpb.ImplantConfig_OutputFormat
	switch format {
	case "exe":
		configFormat = clientpb.ImplantConfig_EXECUTABLE
	case "shared":
		configFormat = clientpb.ImplantConfig_SHARED_LIB
		isSharedLib = true
	case "shellcode":
		configFormat = clientpb.ImplantConfig_SHELLCODE
		isSharedLib = true
	default:
		// default to exe
		configFormat = clientpb.ImplantConfig_EXECUTABLE
	}
	/* For UX we convert some synonymous terms */
	if targetOS == "darwin" || targetOS == "mac" || targetOS == "macos" || targetOS == "m" || targetOS == "osx" {
		targetOS = "darwin"
	}
	if targetOS == "windows" || targetOS == "win" || targetOS == "w" || targetOS == "shit" {
		targetOS = "windows"
	}
	if targetOS == "linux" || targetOS == "unix" || targetOS == "l" {
		targetOS = "linux"
	}
	if arch == "x64" || strings.HasPrefix(arch, "64") {
		arch = "amd64"
	}
	if arch == "x86" || strings.HasPrefix(arch, "32") {
		arch = "386"
	}

	config := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           arch,
		Debug:            ctx.Flags.Bool("debug"),
		ObfuscateSymbols: symbolObfuscation,
		C2:               c2s,
		CanaryDomains:    canaryDomains,

		ReconnectInterval:   uint32(reconnectInterval),
		MaxConnectionErrors: uint32(maxConnectionErrors),

		LimitDomainJoined: limitDomainJoined,
		LimitHostname:     limitHostname,
		LimitUsername:     limitUsername,
		LimitDatetime:     limitDatetime,

		Format:      configFormat,
		IsSharedLib: isSharedLib,
	}

	return config
}

func parseMTLSc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		uri := url.URL{Scheme: "mtls"}
		uri.Host = arg
		if uri.Port() == "" {
			uri.Host = fmt.Sprintf("%s:%d", uri.Host, defaultMTLSLPort)
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseHTTPc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		arg = strings.ToLower(arg)
		var uri *url.URL
		var err error
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			uri, err = url.Parse(arg)
			if err != nil {
				log.Printf("Failed to parse c2 URL %v", err)
				continue
			}
		} else {
			uri = &url.URL{Scheme: "https"} // HTTPS is the default, will fallback to HTTP
			uri.Host = arg
		}
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func parseDNSc2(args string) []*clientpb.ImplantC2 {
	c2s := []*clientpb.ImplantC2{}
	if args == "" {
		return c2s
	}
	for index, arg := range strings.Split(args, ",") {
		uri := url.URL{Scheme: "dns"}
		if len(arg) < 1 {
			continue
		}
		// Make sure we have the FQDN
		if !strings.HasSuffix(arg, ".") {
			arg += "."
		}
		if strings.HasPrefix(arg, ".") {
			arg = arg[1:]
		}

		uri.Host = arg
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(index),
			URL:      uri.String(),
		})
	}
	return c2s
}

func profileGenerate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	name := ctx.Flags.String("name")
	if name == "" && 1 <= len(ctx.Args) {
		name = ctx.Args[0]
	}
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	profiles := getSliverProfiles(rpc)
	if profile, ok := (*profiles)[name]; ok {
		compile(profile.Config, save, rpc)
	} else {
		fmt.Printf(Warn+"No profile with name '%s'", name)
	}
}

func compile(config *clientpb.ImplantConfig, save string, rpc rpcpb.SliverRPCClient) error {

	fmt.Printf(Info+"Generating new %s/%s implant binary\n", config.GOOS, config.GOARCH)

	if config.ObfuscateSymbols {
		fmt.Printf(Info+"%sSymbol obfuscation is enabled.%s\n", bold, normal)
		fmt.Printf(Info + "This process can take awhile, and consumes significant amounts of CPU/Memory\n")
	} else if !config.Debug {
		fmt.Printf(Warn+"Symbol obfuscation is %sdisabled%s\n", bold, normal)
	}

	start := time.Now()
	ctrl := make(chan bool)
	go spin.Until("Compiling, please wait ...", ctrl)

	generated, err := rpc.Generate(context.Background(), &clientpb.GenerateReq{
		Config: config,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return err
	}

	end := time.Now()
	elapsed := time.Time{}.Add(end.Sub(start))
	fmt.Printf(clearln+Info+"Build completed in %s\n", elapsed.Format("15:04:05"))
	if len(generated.File.Data) == 0 {
		fmt.Printf(Warn + "Build failed, no file data\n")
		return errors.New("No file data")
	}

	saveTo, err := saveLocation(save, generated.File.Name)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(saveTo, generated.File.Data, 0500)
	if err != nil {
		fmt.Printf(Warn+"Failed to write to: %s\n", saveTo)
		return err
	}
	fmt.Printf(Info+"Implant saved to %s\n", saveTo)
	return nil
}

func profiles(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	profiles := getSliverProfiles(rpc)
	if profiles == nil {
		return
	}
	if len(*profiles) == 0 {
		fmt.Printf(Info+"No profiles, create one with `%s`\n", consts.NewProfileStr)
		return
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Name\tPlatform\tCommand & Control\tDebug\tLimitations\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Platform")),
		strings.Repeat("=", len("Command & Control")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Limitations")))

	for name, profile := range *profiles {
		config := profile.Config
		if 0 < len(config.C2) {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
				name,
				fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
				fmt.Sprintf("[1] %s", config.C2[0].URL),
				fmt.Sprintf("%v", config.Debug),
				getLimitsString(config),
			)
		}
		if 1 < len(config.C2) {
			for index, c2 := range config.C2[1:] {
				fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
					"",
					"",
					fmt.Sprintf("[%d] %s", index+2, c2.URL),
					"",
					"",
				)
			}
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", "", "", "", "", "")
	}
	table.Flush()
}

func getLimitsString(config *clientpb.ImplantConfig) string {
	limits := []string{}
	if config.LimitDatetime != "" {
		limits = append(limits, fmt.Sprintf("datetime=%s", config.LimitDatetime))
	}
	if config.LimitDomainJoined {
		limits = append(limits, fmt.Sprintf("domainjoined=%v", config.LimitDomainJoined))
	}
	if config.LimitUsername != "" {
		limits = append(limits, fmt.Sprintf("username=%s", config.LimitUsername))
	}
	if config.LimitHostname != "" {
		limits = append(limits, fmt.Sprintf("hostname=%s", config.LimitHostname))
	}
	return strings.Join(limits, "; ")
}

func newProfile(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	name := ctx.Flags.String("name")
	if name == "" {
		fmt.Printf(Warn + "Invalid profile name\n")
		return
	}
	config := parseCompileFlags(ctx)
	if config == nil {
		return
	}
	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := rpc.SaveImplantProfile(context.Background(), profile)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"Saved new profile %s\n", resp.Name)
	}
}

func getSliverProfiles(rpc rpcpb.SliverRPCClient) *map[string]*clientpb.ImplantProfile {
	pbProfiles, err := rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Error %s", err)
		return nil
	}
	profiles := &map[string]*clientpb.ImplantProfile{}
	for _, profile := range pbProfiles.Profiles {
		(*profiles)[profile.Name] = profile
	}
	return profiles
}

func canaries(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	canaries, err := rpc.Canaries(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Failed to list canaries %s", err)
		return
	}
	if 0 < len(canaries.Canaries) {
		displayCanaries(canaries.Canaries, ctx.Flags.Bool("burned"))
	} else {
		fmt.Printf(Info + "No canaries in database\n")
	}
}

func displayCanaries(canaries []*clientpb.DNSCanary, burnedOnly bool) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "Sliver Name\tDomain\tTriggered\tFirst Trigger\tLatest Trigger\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Sliver Name")),
		strings.Repeat("=", len("Domain")),
		strings.Repeat("=", len("Triggered")),
		strings.Repeat("=", len("First Trigger")),
		strings.Repeat("=", len("Latest Trigger")),
	)

	lineColors := []string{}
	for _, canary := range canaries {
		if burnedOnly && !canary.Triggered {
			continue
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			canary.ImplantName,
			canary.Domain,
			fmt.Sprintf("%v", canary.Triggered),
			canary.FirstTriggered,
			canary.LatestTrigger,
		)
		if canary.Triggered {
			lineColors = append(lineColors, bold+red)
		} else {
			lineColors = append(lineColors, normal)
		}
	}
	table.Flush()

	for index, line := range strings.Split(outputBuf.String(), "\n") {
		if len(line) == 0 {
			continue
		}
		// We need to account for the two rows of column headers
		if 0 < len(line) && 2 <= index {
			lineColor := lineColors[index-2]
			fmt.Printf("%s%s%s\n", lineColor, line, normal)
		} else {
			fmt.Printf("%s\n", line)
		}
	}
}
