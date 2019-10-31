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

type pidstats struct {
	pid                   string
	comm                  string
	state                 string
	ppid                  string
	pgrp                  string
	session               string
	tty_nr                string
	tpgid                 string
	flags                 string
	minflt                string
	cminflt               string
	majflt                string
	cmajflt               string
	utime                 string
	stime                 string
	cutime                string
	cstime                string
	priority              string
	nice                  string
	num_threads           string
	itrealvalue           string
	starttime             string
	vsize                 string
	rss                   string
	rsslim                string
	startcode             string
	encode                string
	startstack            string
	kstkesp               string
	kstkeip               string
	signal                string
	blocked               string
	sigignore             string
	sigcatch              string
	wchan                 string
	nswap                 string
	cnswap                string
	exit_signal           string
	processor             string
	rt_priority           string
	policy                string
	delayacct_blkio_ticks string
	guest_time            string
	cguest_time           string
	start_data            string
	end_data              string
	start_brk             string
	arg_start             string
	arg_end               string
	env_start             string
	env_end               string
	exit_code             string
}

var pids map[int]pidstats

func main() {
	pids = make(map[int]pidstats)
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/login", login)
	http.HandleFunc("/monitor", monitor)
	http.ListenAndServe(":9000", nil) // Set port

}

func login(response http.ResponseWriter, request *http.Request) {
	fmt.Println("method:", request.Method) // Get request method

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

	/*
		if request.Method == "GET" {
			t, _ := template.ParseFiles("index.html")
			t.Execute(response, nil)
		} else {
			request.ParseForm()
			var username string = fmt.Sprint(request.Form["username"][0])
			var password string = fmt.Sprint(request.Form["password"][0])
			var ip string = fmt.Sprint(request.Form["ip"][0])
			connect(username, password, ip)

			//output := []byte("Temporary Empty Page. . . ")
			//response.Write(output)
			//fmt.Println("Page is loading")
		}
	*/
}

func monitor(response http.ResponseWriter, request *http.Request) {
	//output := []byte("Temporary Empty Page. . . ")
	//response.Write(output)

	fmt.Println("method:", request.Method) // Get request method

	switch request.Method {
	case "GET":
		t, _ := template.ParseFiles("monitor.html")
		t.Execute(response, nil)
	case "POST":
		fmt.Println("What am I even posting")
	}
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

		//getCPUInfo(in, out)
		getMemInfo(in, out)
		//getUptime(in, out)
		//getProcesses(in, out)
		//printMap()

		// automatically "exit"
		in <- "exit"
		session.Wait()
		http.Redirect(response, request, "/monitor", http.StatusSeeOther)
	}
}

func printMap() {
	for _, v := range pids {
		fmt.Printf("%+v", v)
		fmt.Print("\n\n")
	}
}

func getCPUInfo(in chan<- string, out <-chan string) {
	in <- "cat /proc/cpuinfo"
	out2Slice(out, "\n")
}

func getMemInfo(in chan<- string, out <-chan string) {
	in <- "cat /proc/meminfo"
	out2Slice(out, "\n")
}

func getUptime(in chan<- string, out <-chan string) {
	in <- "cat /proc/uptime"
	out2Slice(out, " ")
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
			pids[v] = pidstats{
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

//func out2Slice(out <-chan string, split string) []string {
func out2Slice(out <-chan string, split string) {
	//var s string = <-out
	slice := strings.Split(<-out, split)
	for i, v := range slice {
		fmt.Print(strconv.Itoa(i) + ") ")
		fmt.Println(v)
	}
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
