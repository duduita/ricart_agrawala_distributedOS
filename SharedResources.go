package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
		os.Exit(0)
	}
}

func doCSServerJob(CSConn *net.UDPConn) {
	//Loop infinito mesmo
	buf := make([]byte, 1024)

	for {
		//Ler (uma vez somente) da conexão UDP a mensagem
		//Escrever na tela a msg recebida (indicando o
		//endereço de quem enviou)
		n, addr, err := CSConn.ReadFromUDP(buf)
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)

		if err != nil {
			fmt.Println("Error: ", err)
		}

	}
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":10000")
	CheckError(err)
	CSConn, err := net.ListenUDP("udp", addr)
	CheckError(err)
	defer CSConn.Close()
	for {
		go doCSServerJob(CSConn)
		//Loop infinito para receber mensagem e escrever todo
		//conteúdo (processo que enviou, relógio recebido e texto)
		//na tela
		//FALTA FAZER
		time.Sleep(time.Second * 1)

	}
}
