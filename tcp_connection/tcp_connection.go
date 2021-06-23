package tcp_connection

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	readDeadLine  = 2
	writeDeadLine = 2
)

type Lamp struct {
	Ip             string
	Port           string
	Mqtt_topic_ctl string
	Mqtt_topic_sts string
	Lamp_name      string
	Conn           net.Conn
}

type LampPool struct {
	Pool map[string]*Lamp
}

func InitLampPool() LampPool {
	var lamp_pool LampPool
	lamp_pool.Pool = make(map[string]*Lamp)
	return lamp_pool
}

func AddLamp(lamp_pool *LampPool, lamp_name, ip, port, mqtt_topic_ctl, mqtt_topic_sts string) {
	var con Lamp
	con.Lamp_name = lamp_name
	con.Ip = ip
	con.Port = port
	con.Mqtt_topic_ctl = mqtt_topic_ctl
	con.Mqtt_topic_sts = mqtt_topic_sts
	lamp_pool.Pool[lamp_name] = &con
}

func GetLampByCtlTopic(lamp_pool *LampPool, mqtt_topic_ctl string) string {
	for key, lamp := range lamp_pool.Pool {
		if lamp.Mqtt_topic_ctl == mqtt_topic_ctl {
			return key
		}
	}
	return ""
}

func ConnectLamp(lamp *Lamp) bool {
	//fmt.Printf("[%s] ConnectLamp: Connecting to Name=%s, Host=%s:%s\n", time.Now().Format("15:04:05.000"), lamp.Lamp_name, lamp.Ip, lamp.Port)
	var err error
	var tcpAddr *net.TCPAddr
	if tcpAddr, err = net.ResolveTCPAddr("tcp4", lamp.Ip+":"+lamp.Port); err != nil {
		lamp.Conn = nil
		log.WithFields(log.Fields{"modul": "tcp"}).Error("Can not resolve IP for lamp:" + lamp.Lamp_name + ". " + err.Error())
		//fmt.Println("Can not resolve IP for lamp:" + lamp.Lamp_name + ". " + err.Error())
		return false
	}

	if lamp.Conn, err = net.DialTCP("tcp", nil, tcpAddr); err != nil {
		lamp.Conn = nil
		log.WithFields(log.Fields{"modul": "tcp"}).Error("Can not connect to lamp:" + lamp.Lamp_name + ". " + err.Error())
		//fmt.Println("Can not connect to lamp:" + lamp.Lamp_name + ". " + err.Error())
		return false
	}

	log.WithFields(log.Fields{"modul": "tcp"}).Info("Connection established to lamp: " + lamp.Lamp_name + ", " + lamp.Ip + ":" + lamp.Port + "]")
	fmt.Printf("[%s] ConnectLamp: Connection established for Name=%s, Host=%s:%s\n", time.Now().Format("15:04:05.000"), lamp.Lamp_name, lamp.Ip, lamp.Port)
	return true
}

func SendCommandLamp(lamp *Lamp, command string) (string, bool) {
	var buf [512]byte
	var err error
	var n int

	if lamp.Conn == nil {
		log.WithFields(log.Fields{"modul": "tcp"}).Warning("There is no active connection to lamp:" + lamp.Lamp_name)
		//fmt.Println("There is no active connection to lamp:" + lamp.Lamp_name)
		return "", false
	}

	lamp.Conn.SetWriteDeadline(time.Now().Add(time.Second * writeDeadLine))
	if _, err = lamp.Conn.Write([]byte(command + "\r\n")); err != nil {
		log.WithFields(log.Fields{"modul": "tcp"}).Warning("Can not write to lamp:" + lamp.Lamp_name + ". " + err.Error())
		//fmt.Println("Can not write to lamp:" + lamp.Lamp_name + ". " + err.Error())
		return "", false
	}

	lamp.Conn.SetReadDeadline(time.Now().Add(time.Second * readDeadLine))
	if n, err = lamp.Conn.Read(buf[0:]); err != nil {
		log.WithFields(log.Fields{"modul": "tcp"}).Warning("Can not Read to lamp:" + lamp.Lamp_name + ". " + err.Error())
		//fmt.Println("Can not Read to lamp:" + lamp.Lamp_name + ". " + err.Error())
		return "", false
	}
	return string(buf[0:n]), true
}

// TODO: Need to make handler of TCP connection
func ConnectionHadler(lamp_pool *LampPool) {
	defer func() {
		//fmt.Printf("[%s] ConnectionHadler: closed \n", time.Now().Format("15:04:05.000"))
		log.WithFields(log.Fields{"modul": "tcp"}).Info("ConnectionHadler: Closed: " + lamp_pool.Pool["yeelight1"].Ip)
	}()

	fmt.Printf("[%s] connectionHadler: started\n", time.Now().Format("15:04:05.000"))
	for {
		fmt.Printf("[%s] connectionHadler: Lamp = %s \n", time.Now().Format("15:04:05.000"), lamp_pool.Pool["yeelight1"].Lamp_name)
		fmt.Printf("[%s] connectionHadler: IP   = %s \n", time.Now().Format("15:04:05.000"), lamp_pool.Pool["yeelight1"].Ip)
		time.Sleep(20 * time.Second)
	}
}
