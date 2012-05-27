package main

import (
	"bufio"
	"code.google.com/p/go-netrc/netrc"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	Version = "0.0.1"
)

var (
	apiURL = "https://api.heroku.com"
	hkHome = os.Getenv("HOME") + "/.hk"
)

var stdin = bufio.NewReader(os.Stdin)

var updater = Updater{
	url: "https://github.com/downloads/kr/hk/",
	dir: hkHome + "/update/",
}

type Command struct {
	// args does not include the command name
	Run func(cmd *Command, args []string)

	Usage string // first word is the command name
	Short string // `hk help` output
	Long  string // `hk help <cmd>` output
}

func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// Running `hk help` will list commands in this order.
var commands = []*Command{
	cmdCreds,
	cmdEnv,
	cmdFetchUpdate,
	cmdGet,
	cmdInfo,
	cmdList,
	cmdPs,
	cmdVersion,
	cmdHelp,
}

var flagApp = flag.String("a", "", "app")

func main() {
	defer updater.run() // doesn't run if os.Exit is called

	if s := os.Getenv("HEROKU_API_URL"); s != "" {
		apiURL = strings.TrimRight(s, "/")
	}

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	name := args[0]
	os.Args = args
	flag.Parse()
	args = flag.Args()

	for _, cmd := range commands {
		if cmd.Name() == name {
			cmd.Run(cmd, args)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown command: %s\n", name)
	usage()
}

func getCreds(u *url.URL) (user, pass string) {
	if u.User != nil {
		pw, _ := u.User.Password()
		return u.User.Username(), pw
	}

	m, err := netrc.FindMachine(os.Getenv("HOME")+"/.netrc", u.Host)
	if err != nil {
		panic(err)
	}

	return m.Login, m.Password
}

func apiReq(v interface{}, meth string, url string) {
	req, err := http.NewRequest(meth, url, nil)
	if err != nil {
		panic(err)
	}

	req.SetBasicAuth(getCreds(req.URL))
	req.Header.Add("User-Agent", fmt.Sprintf("hk/%s", Version))
	req.Header.Add("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	if res.StatusCode == 401 {
		errorf("Unauthorized")
	}
	if res.StatusCode == 403 {
		errorf("Unauthorized")
	}
	if res.StatusCode != 200 {
		fmt.Printf("%v\n", res)
		errorf("Unexpected error")
	}

	err = json.NewDecoder(res.Body).Decode(v)
	if err != nil {
		panic(err)
	}
}

func errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format, a...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func unrecArg(arg, cmd string) {
	errorf("Unrecognized argument '%s'. See 'hk help %s'", arg, cmd)
}