package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// CompStats =
type CompStats struct {
	ModelName   string
	CPUMhz      string
	CacheSize   string
	MemTotal    string
	MemFree     string
	SwapCached  string
	FirstLoad   string
	SecondLoad  string
	ThirdLoad   string
	TotalLoad   string
	UpTime      string
	IdleTime    string
	CurrentUser string
	HostName    string
	Chassis     string
	Operating   string
	Kernel      string
}

// PidStats =
type PidStats struct {
	Pid         string
	Cmd         string
	State       string
	Ppid        string
	Pgrp        string
	Session     string
	TtyNr       string
	Tpgid       string
	Flags       string
	Minflt      string
	Cminflt     string
	Majflt      string
	Cmajflt     string
	Utime       string
	Stime       string
	Cutime      string
	Cstime      string
	Priority    string
	Nice        string
	NumThreads  string
	Itrealvalue string
	Starttime   string
	Vsize       string
	Rss         string
	Rsslim      string
	Startcode   string
	Encode      string
	Startstack  string
	Kstkesp     string
	Kstkeip     string
	Signal      string
	Blocked     string
	Sigignore   string
	Sigcatch    string
	Wchan       string
	Nswap       string
	Cnswap      string
	ExitSignal  string
	Processor   string
	RtPriority  string
	Policy      string
	Delays      string
	GuestTime   string
	CguestTime  string
	StartData   string
	EndData     string
	StartBrk    string
	ArgStart    string
	ArgEnd      string
	EnvStart    string
	EnvEnd      string
	ExitCode    string
}

var computer CompStats

// Pids =
var Pids map[int]PidStats

func main() {
	Pids = make(map[int]PidStats)
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/login", login)
	http.HandleFunc("/monitor", monitor)
	http.ListenAndServe(":9000", nil) // Set port

}

func login(response http.ResponseWriter, request *http.Request) {
	fmt.Println("login METHOD:", request.Method)

	switch request.Method {
	case "GET":
		t, _ := template.ParseFiles("index.html")
		t.Execute(response, nil)
	case "POST":
		request.ParseForm()
		var username string = fmt.Sprint(request.Form["username"][0])
		var password string = fmt.Sprint(request.Form["password"][0])
		var ip string = fmt.Sprint(request.Form["ip"][0])
		var port string = fmt.Sprint(request.Form["port"][0])
		connect(username, password, ip, port, response, request)
	}
}

func monitor(response http.ResponseWriter, request *http.Request) {
	fmt.Println("monitor METHOD:", request.Method)

	html := `
	<p>
	{{.CurrentUser}}@{{.HostName}}<br>
	Model Name: {{.ModelName}}<br>
	Operating System: {{.Operating}}<br>
	Kernel: {{.Kernel}}<br>
	Chassis: {{.Chassis}}<br>
	<br>
	Cache Size: {{.CacheSize}}<br>
	Swap Cache: {{.SwapCached}}<br>
	RAM Memory: {{.MemFree}}/{{.MemTotal}}<br>
	<br>
	Tasks: {{.TotalLoad}}<br>
	Load Average: {{.FirstLoad}} {{.SecondLoad}} {{.ThirdLoad}}<br>
	</p>
	`

	html2 := `
	<table>
	<tr>
		<td>PID</td>
		<td>CMD</td>
		<td>State</td>
		<td>PPID</td>
		<td>Priority</td>
		<td>Nice</td>
		<td>VIRT</td>
		<td>RES</td>
		<td>Session</td>
		<td>EnvStart</td>
	</tr>
	{{range $key, $value := .}}
	<tr>
		<td>{{$value.Pid}}</td>
		<td>{{$value.Cmd}}</td>
		<td>{{$value.State}}</td>
		<td>{{$value.Ppid}}</td>
		<td>{{$value.Priority}}</td>
		<td>{{$value.Nice}}</td>
		<td>{{$value.Vsize}}</td>
		<td>{{$value.Rss}}</td>
		<td>{{$value.Session}}</td>
		<td>{{$value.EnvStart}}</td>
	</tr>
	{{end}}
	`

	template1 := template.Must(template.New("").Parse(html))
	template1.Execute(response, computer)

	template2 := template.Must(template.New("").Parse(html2))
	template2.Execute(response, Pids)
}

func connect(username string, password string, ip string, port string, response http.ResponseWriter, request *http.Request) {
	fmt.Println("Username:", username)
	fmt.Println("Password:", password)
	fmt.Println("IP Address:", ip)
	fmt.Println("Port:", port)
	fmt.Println()

	// Connect to ssh cient
	config := &ssh.ClientConfig{
		//To resolve "Failed to dial: ssh: must specify HostKeyCallback" error
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}
	client, err := ssh.Dial("tcp", ip+":"+port, config)
	if err != nil {
		//panic("Dial Failed: " + err.Error())
		fmt.Println("Invalid Login or Password")
		http.Redirect(response, request, "/login", http.StatusSeeOther)
	} else {

		session, err := client.NewSession()
		if err != nil {
			panic("Session Failed: " + err.Error())
		}
		defer session.Close()

		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		if err := session.RequestPty("xterm", 80, 100, modes); err != nil {
			log.Fatal(err)
		}

		fmt.Println("///////////////////////////////")
		fmt.Println("Successfully Connected!")
		fmt.Println("///////////////////////////////")
		fmt.Println()

		w, err := session.StdinPipe()
		if err != nil {
			panic(err)
		}
		r, err := session.StdoutPipe()
		if err != nil {
			panic(err)
		}
		in, out := MuxShell(w, r)
		if err := session.Start("/bin/sh"); err != nil {
			log.Fatal(err)
		}

		// Ignore the shell output
		<-out

		/*
			in <- "cat file1.txt" + "&&" + "cat file4"
			fmt.Println(<-out)

			in <- "whoami"
			fmt.Println(<-out)
		*/

		getCPUInfo(in, out)
		getMemInfo(in, out)
		getUptime(in, out)
		getLoadAvg(in, out)
		getUser(in, out)
		getHostInfo(in, out)
		getProcesses(in, out)

		fmt.Printf("%+v\n", computer)
		fmt.Println()
		//printMap()

		// automatically "exit"
		in <- "exit"
		session.Wait()
		http.Redirect(response, request, "/monitor", http.StatusSeeOther)
	}
}

func printMap() {
	for _, v := range Pids {
		fmt.Printf("%+v", v)
		fmt.Print("\n\n")
	}
}

func getCPUInfo(in chan<- string, out <-chan string) {
	in <- "cat /proc/cpuinfo | egrep 'model name|cpu MHz|cache size'"
	slice := out2Slice(out, "\n")
	hold := parseSemicolon(slice)
	computer.ModelName = hold[0]
	computer.CPUMhz = hold[1]
	computer.CacheSize = hold[2]
}

func getMemInfo(in chan<- string, out <-chan string) {
	in <- "cat /proc/meminfo | egrep 'MemTotal|MemFree|SwapCached'"
	slice := out2Slice(out, "\n")
	hold := parseSemicolon(slice)
	computer.MemTotal = hold[0]
	computer.MemFree = hold[1]
	computer.SwapCached = hold[2]
}

func getUptime(in chan<- string, out <-chan string) {
	in <- "cat /proc/uptime"
	slice := strings.Fields(<-out)
	computer.UpTime = slice[0]
	computer.IdleTime = slice[1]
}

func getLoadAvg(in chan<- string, out <-chan string) {
	in <- "cat /proc/loadavg"
	slice := out2Slice(out, " ")
	computer.FirstLoad = slice[0]
	computer.SecondLoad = slice[1]
	computer.ThirdLoad = slice[2]
	computer.TotalLoad = slice[3]
}

func getUser(in chan<- string, out <-chan string) {
	in <- "whoami"
	slice := strings.Fields(<-out)
	computer.CurrentUser = slice[0]
}

func getHostInfo(in chan<- string, out <-chan string) {
	in <- "hostnamectl | egrep 'Static hostname|Chassis|Operating System|Kernel'"
	slice := out2Slice(out, "\n")
	hold := parseSemicolon(slice)
	computer.HostName = hold[0]
	computer.Chassis = hold[1]
	computer.Operating = hold[2]
	computer.Kernel = hold[3]
}

func getProcesses(in chan<- string, out <-chan string) {
	in <- "ls /proc"

	slice := []int{}

	// Try to convert listed directories names to "int"
	// If it works then it's a process directory else skip
	temp := strings.Fields(<-out)
	for _, v := range temp {
		hold, err := strconv.Atoi(v)
		if err == nil {
			slice = append(slice, hold)
		}
	}

	for _, v := range slice {
		cmd := "cat /proc/" + strconv.Itoa(v) + "/stat"
		in <- cmd
		data := strings.Fields(<-out)

		if data[0] != "cat:" {
			Pids[v] = PidStats{
				data[0], data[1], data[2], data[3], data[4], data[5], data[6],
				data[7], data[8], data[9], data[10], data[11], data[12], data[13],
				data[14], data[15], data[16], data[17], data[18], data[19], data[20],
				data[21], data[22], data[23], data[24], data[25], data[26], data[27],
				data[28], data[29], data[30], data[31], data[32], data[33], data[34],
				data[35], data[36], data[37], data[38], data[39], data[40], data[41],
				data[42], data[43], data[44], data[45], data[46], data[47], data[48],
				data[49], data[50], data[51],
			}
		}
	}
}

func out2Slice(out <-chan string, split string) []string {
	//var s string = <-out
	slice := strings.Split(<-out, split)
	for i, v := range slice {
		fmt.Print(strconv.Itoa(i) + ") ")
		fmt.Println(v)
	}
	return slice
}

func parseSemicolon(slice []string) []string {
	var hold []string
	for i := 0; i < len(slice)-1; i++ {
		s := strings.Split(slice[i], ":")
		hold = append(hold, strings.TrimSpace(s[1]))
	}
	return hold
}

// MuxShell =
func MuxShell(w io.Writer, r io.Reader) (chan<- string, <-chan string) {
	in := make(chan string, 1)
	out := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1) //for the shell itself
	go func() {
		for cmd := range in {
			wg.Add(1)
			w.Write([]byte(cmd + "\n"))
			wg.Wait()
		}
	}()
	go func() {
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			n, err := r.Read(buf[t:])
			if err != nil {
				close(in)
				close(out)
				return
			}
			t += n
			if buf[t-2] == '$' { //assuming the $PS1 == 'sh-4.3$ '
				out <- string(buf[:t])
				t = 0
				wg.Done()
			}
		}
	}()
	return in, out
}
