package main

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
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
var mutex sync.Mutex

func obtain_max(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}
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
func doCSJob() {
	// Usa a CS
	fmt.Println("Entrei na CS")
	myId_str := strconv.Itoa(myId)
	myClock_str := strconv.Itoa(myClock)
	cs_msg := myId_str + " is in the CS now!"
	cs_buf := []byte(cs_msg)
	_, err := ConnMap[idMap[0]].Write(cs_buf)
	CheckError(err)
	mutex.Lock()
	process_state = "HELD"
	mutex.Unlock()
	fmt.Println("HELD")
	time.Sleep(time.Second * 10)

	// Após utilizar a CS, envia REPLY para todos da sua fila
	fmt.Println("Enviando reply para todos")
	rep_msg_all := myId_str + ":" + myClock_str + ":" + "REPLY"
	rep_buf_all := []byte(rep_msg_all)
	if len(queue) > 0 {
		for _, awaiting_processess := range queue {
			ConnMap[idMap[awaiting_processess]].Write(rep_buf_all)
		}
	}

	// Reinicializa o processo
	mutex.Lock()
	reply_counter = 0
	queue = nil
	process_state = "RELEASED"
	mutex.Unlock()
	fmt.Println("RELEASED")
	fmt.Println("Sai da CS")
}
func doServerJob() {
	buf := make([]byte, 1024)

	for {
		// leitura da mensagem
		n, _, err := ServConn.ReadFromUDP(buf)
		received_msg := strings.Split(string(buf[0:n]), ":")
		fmt.Println("Received:", received_msg[2], "clock:", received_msg[1], "from:", received_msg[0])
		received_clock, _ := strconv.Atoi(received_msg[1])
		received_id, _ := strconv.Atoi(received_msg[0])

		// se receber um reply
		if received_msg[2] == "REPLY" {
			mutex.Lock()
			reply_counter++
			mutex.Unlock()

			// Quando receber reply, atualizar o clock para ficar coerente com quem enviou
			receivedClock, _ := strconv.Atoi(string(buf[0:n]))
			mutex.Lock()
			myClock = obtain_max(myClock, receivedClock) + 1
			mutex.Unlock()
			fmt.Println("My new clock after receive a reply: ", myClock)

			// se receber reply de todo mundo, entra na CS
			if reply_counter == nServers-1 {
				go doCSJob()
			}
			// Se receber um request de si mesmo
		} else if received_id == myId {
			fmt.Println("Don't need reply, it's just a intern action")

			// Quando houver uma ação interna, incrementar o clock
			mutex.Lock()
			myClock++
			mutex.Unlock()
			fmt.Println("My new clock after an intern action: ", myClock)
			CheckError(err)
			// Se estiver HELD ou WANTED com clock menor, adiciona em sua fila
		} else if process_state == "HELD" || process_state == "WANTED" && received_clock > myClock {
			mutex.Lock()
			queue = append(queue, received_id)
			mutex.Unlock()
			fmt.Println("Queue request without replying")
			fmt.Println(queue)

			// Quando receber um request, atualizar o clock para ficar coerente com quem enviou
			receivedClock, _ := strconv.Atoi(string(buf[0:n]))
			mutex.Lock()
			myClock = obtain_max(myClock, receivedClock) + 1
			mutex.Unlock()
			fmt.Println("My new clock after request (a): ", myClock)
		} else {
			// se empatarem no clock, ganha o menor id
			priority_id := received_id
			if received_clock == myClock {
				priority_id = int(math.Min(float64(received_id), float64(myId)))
			}

			mutex.Lock()
			myClock = obtain_max(myClock, received_clock) + 1
			mutex.Unlock()
			// Quando enviar um reply, não incrementar o clock
			fmt.Println("My new clock after request (b): ", myClock)
			fmt.Println("Reply due to RELEASED or WANTED with lower timestamp")
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
	if otherProcess == 0 && process_state == "RELEASED" {
		// Altera de RELEASED para WANTED
		mutex.Lock()
		process_state = "WANTED"
		myClock++
		mutex.Unlock()
		fmt.Println("WANTED")
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
	} else {
		fmt.Println("entrada ignorada!")
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
	mutex.Lock()
	process_state = "RELEASED"
	myClock = 0
	myId, _ = strconv.Atoi(os.Args[1])
	myPort = ":" + strconv.Itoa(idMap[myId])
	mutex.Unlock()

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
