package main

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processo
var CliConn []*net.UDPConn //vetor com conexões para os servidores dos outros processos
var ServConn *net.UDPConn  //conexão do meu servidor (onde recebo mensagens dos outros processos)
var myId int
var myClock int
var ConnMap map[int]*net.UDPConn
var idMap map[int]int
var process_state string
var queue []int
var reply_counter int

func readInput(ch chan string) {
	// Rotina não-bloqueante que “escuta” o stdin
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}
func CheckError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}
func PrintError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
	}
}
func doServerJob() {
	buf := make([]byte, 1024)

	for {
		// leitura da mensagem
		n, addr, err := ServConn.ReadFromUDP(buf)
		received_msg := strings.Split(string(buf[0:n]), ":")
		fmt.Println("Received id:", received_msg[0], "clock:", received_msg[1], "message:", received_msg[2], "from ", addr)
		received_clock, _ := strconv.Atoi(received_msg[1])
		received_id, _ := strconv.Atoi(received_msg[0])

		if received_msg[2] == "REPLY" {
			reply_counter++
			if reply_counter == nServers-1 {
				process_state = "HELD"
				time.Sleep(time.Second * 1)
				reply_counter = 0
				myId_str := strconv.Itoa(myId)
				myClock_str := strconv.Itoa(myClock)
				rep_msg := myId_str + ":" + myClock_str + ":" + "REPLY"
				rep_buf := []byte(rep_msg)
				for awaiting_processess := range queue {
					ConnMap[idMap[awaiting_processess]].Write(rep_buf)
				}
				queue = nil
				process_state = "RELEASED"
			}
		}

		if received_msg[2] == "REQUEST" && process_state == "RELEASED" || received_clock <= myClock && process_state == "WANTED" || process_state == "HELD" {
			priority_id := received_id
			if received_clock == myClock {
				priority_id = int(math.Min(float64(received_id), float64(myId)))
			}
			fmt.Println("Need reply")
			myId_str := strconv.Itoa(myId)
			myClock_str := strconv.Itoa(myClock)
			rep_msg := myId_str + ":" + myClock_str + ":" + "REPLY"
			rep_buf := []byte(rep_msg)
			_, err := ConnMap[idMap[priority_id]].Write(rep_buf)
			if err != nil {
				fmt.Println(rep_msg, err)
			}
		} else {
			queue = append(queue, received_id)
			fmt.Println("Don't need reply, so queue it")
			receivedClock, _ := strconv.Atoi(string(buf[0:n]))
			myClock = int(math.Max(float64(myClock), float64(receivedClock)) + 1)
			fmt.Println("My new clock is: ", myClock)
			if err != nil {
				fmt.Println("Error: ", err)
			}
		}
	}
}
func doClientJob(otherProcess int, type_message string, x string) {
	//Enviar uma mensagem (com valor i) para o servidor do processo
	//otherServer

	if otherProcess == 0 {
		process_state = "WANTED"
		myId_str := strconv.Itoa(myId)
		myClock_str := strconv.Itoa(myClock)
		cs_msg := myId_str + ":" + myClock_str + ":" + "hello world"
		req_msg := myId_str + ":" + myClock_str + ":" + "REQUEST"
		cs_buf := []byte(cs_msg)
		req_buf := []byte(req_msg)
		for dest_ids := range idMap {
			ConnMap[idMap[dest_ids]].Write(req_buf)
		}
		_, err := ConnMap[idMap[otherProcess]].Write(cs_buf)
		if err != nil {
			fmt.Println(cs_msg, err)
		}
	} else {
		myId_str := strconv.Itoa(myId)
		myClock_str := strconv.Itoa(myClock)
		req_msg := myId_str + ":" + myClock_str + ":" + "REQUEST"
		req_buf := []byte(req_msg)
		_, err := ConnMap[idMap[otherProcess]].Write(req_buf)
		if err != nil {
			fmt.Println(req_msg, err)
		}
	}

	time.Sleep(time.Second * 1)
}

func initConnections() {
	ConnMap = make(map[int]*net.UDPConn)
	idMap = make(map[int]int)
	idMap[0] = 10000
	idMap[1] = 10002
	idMap[2] = 10003
	idMap[3] = 10004
	process_state = "RELEASED"
	myClock = 0
	myId, _ = strconv.Atoi(os.Args[1])
	myPort = ":" + strconv.Itoa(idMap[myId])
	nServers = len(os.Args) - 2
	/*Esse 2 tira o nome (no caso Process) e tira a primeira porta (que
	é a minha). As demais portas são dos outros processos*/
	// CliConn = make([]*net.UDPConn, nServers)

	/*Outros códigos para deixar ok a conexão do meu servidor (onde re-
	cebo msgs). O processo já deve ficar habilitado a receber msgs.*/

	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	CheckError(err)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)
	/*Outros códigos para deixar ok as conexões com os servidores dos
	outros processos. Colocar tais conexões no vetor CliConn.*/

	for servidores := 0; servidores < nServers; servidores++ {
		ServerAddr, err := net.ResolveUDPAddr("udp",
			"127.0.0.1"+os.Args[2+servidores])
		CheckError(err)
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		port_number_str := os.Args[2+servidores]
		port_number, _ := strconv.Atoi(port_number_str[1:])
		ConnMap[port_number] = Conn
		// CliConn[servidores] = Conn
		CheckError(err)
	}
	cs_port := strconv.Itoa(idMap[0])
	ServerAddr, err = net.ResolveUDPAddr("udp",
		"127.0.0.1"+":"+cs_port)
	CheckError(err)
	Conn, err := net.DialUDP("udp", nil, ServerAddr)
	ConnMap[idMap[0]] = Conn
	// CliConn[servidores] = Conn
	CheckError(err)
}
func main() {
	initConnections()
	/*O fechamento de conexões deve ficar aqui, assim só fecha
	conexão quando a main morrer*/
	defer ServConn.Close()
	for _, connection := range ConnMap {
		defer connection.Close()
	}

	/*Todo Process fará a mesma coisa: ficar ouvindo mensagens e man-
	dar infinitos i’s para os outros processos*/

	ch := make(chan string) //canal que guarda itens lidos do teclado
	go readInput(ch)        //chamar rotina que ”escuta” o teclado
	go doServerJob()
	for {
		// Verificar (de forma não bloqueante) se tem algo no
		// stdin (input do terminal)
		select {
		case x, valid := <-ch:
			if valid {
				fmt.Printf("Recebi do teclado: %s \n", x)
				dest_id, input_error := strconv.Atoi(x)
				// CheckError(input_error)
				if input_error == nil {
					go doClientJob(dest_id, "test", x)
				} else {
					fmt.Printf("Caractere inválido!!\n")
				}
			} else {
				fmt.Println("Canal fechado!")
			}
		default:
			// Fazer nada...
			// Mas não fica bloqueado esperando o teclado
			time.Sleep(time.Second * 1)
		}
		// Esperar um pouco
		time.Sleep(time.Second * 1)

	}
}
