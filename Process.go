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

		// se receber um reply
		if received_msg[2] == "REPLY" {
			reply_counter++

			// Quando receber reply, atualizar o clock para ficar coerente com quem enviou
			receivedClock, _ := strconv.Atoi(string(buf[0:n]))
			myClock = int(math.Max(float64(myClock), float64(receivedClock)) + 1)
			fmt.Println("My new clock is: ", myClock)

			// se receber reply de todo mundo, entra na CS
			if reply_counter == nServers-1 {

				// Usa a CS
				myId_str := strconv.Itoa(myId)
				myClock_str := strconv.Itoa(myClock)
				cs_msg := myId_str + ":" + myClock_str + ":" + "i'am in the CS now"
				cs_buf := []byte(cs_msg)
				_, err := ConnMap[idMap[0]].Write(cs_buf)
				CheckError(err)
				process_state = "HELD"
				time.Sleep(time.Second * 1)

				// Após utilizar a CS, envia REPLY para todos da sua fila
				rep_msg := myId_str + ":" + myClock_str + ":" + "REPLY"
				rep_buf := []byte(rep_msg)
				for awaiting_processess := range queue {
					ConnMap[idMap[awaiting_processess]].Write(rep_buf)
				}

				// Reinicializa o processo
				reply_counter = 0
				queue = nil
				process_state = "RELEASED"
			}
			// Se receber um request de si mesmo
		} else if received_id == myId {
			fmt.Println("Don't need reply, it's just a intern action")

			// Quando houver uma ação interna, incrementar o clock
			myClock++
			fmt.Println("My new clock is: ", myClock)
			CheckError(err)
			// Se estiver HELD ou WANTED com clock menor, adiciona em sua fila
		} else if process_state == "HELD" || process_state == "WANTED" && received_clock > myClock {
			queue = append(queue, received_id)
			fmt.Println("Queue request without replying")

			// Quando receber um request, atualizar o clock para ficar coerente com quem enviou
			receivedClock, _ := strconv.Atoi(string(buf[0:n]))
			myClock = int(math.Max(float64(myClock), float64(receivedClock)) + 1)
			fmt.Println("My new clock is: ", myClock)
		} else {
			// se empatarem no clock, ganha o menor id
			priority_id := received_id
			if received_clock == myClock {
				priority_id = int(math.Min(float64(received_id), float64(myId)))
			}

			// Quando enviar um reply, não incrementar o clock
			fmt.Println("RELEASED or WANTED with lower timestamp: need reply")
			myId_str := strconv.Itoa(myId)
			myClock_str := strconv.Itoa(myClock)
			rep_msg := myId_str + ":" + myClock_str + ":" + "REPLY"
			rep_buf := []byte(rep_msg)
			_, err := ConnMap[idMap[priority_id]].Write(rep_buf)
			CheckError(err)
		}
	}
}
func doClientJob(otherProcess int, type_message string, x string) {

	// para entrar na CS
	if otherProcess == 0 {
		// Altera de RELEASED para WANTED
		process_state = "WANTED"
		myClock++
		myId_str := strconv.Itoa(myId)
		myClock_str := strconv.Itoa(myClock)
		req_msg := myId_str + ":" + myClock_str + ":" + "REQUEST"
		req_buf := []byte(req_msg)

		// Multicast request para todos processos
		for dest_ids := range idMap {
			if dest_ids != 0 && dest_ids != myId {
				ConnMap[idMap[dest_ids]].Write(req_buf)
			}
		}

		time.Sleep(time.Second * 1)
	}
}

func initConnections() {
	// Inicializando variáveis globais
	ConnMap = make(map[int]*net.UDPConn)
	idMap = make(map[int]int)
	idMap[0] = 10000
	idMap[1] = 10002
	idMap[2] = 10003
	idMap[3] = 10004

	// Inicializando o processo
	process_state = "RELEASED"
	myClock = 0
	myId, _ = strconv.Atoi(os.Args[1])
	myPort = ":" + strconv.Itoa(idMap[myId])

	// Obtendo os outros processos
	nServers = len(os.Args) - 2
	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	CheckError(err)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)

	// Gerando o mapa de connections
	for servidores := 0; servidores < nServers; servidores++ {
		ServerAddr, err := net.ResolveUDPAddr("udp",
			"127.0.0.1"+os.Args[2+servidores])
		CheckError(err)
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		port_number_str := os.Args[2+servidores]
		port_number, _ := strconv.Atoi(port_number_str[1:])
		ConnMap[port_number] = Conn
		CheckError(err)
	}

	// Adicionando o sharedResource idMap[0]
	cs_port := strconv.Itoa(idMap[0])
	ServerAddr, err = net.ResolveUDPAddr("udp",
		"127.0.0.1"+":"+cs_port)
	CheckError(err)
	Conn, err := net.DialUDP("udp", nil, ServerAddr)
	ConnMap[idMap[0]] = Conn
	CheckError(err)
}
func main() {
	initConnections()
	defer ServConn.Close()
	for _, connection := range ConnMap {
		defer connection.Close()
	}

	ch := make(chan string) //canal que guarda itens lidos do teclado
	go readInput(ch)        //chamar rotina que ”escuta” o teclado
	go doServerJob()
	for {
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
