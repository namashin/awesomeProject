package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/jlaffaye/ftp"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

import "C"

// ErrFileExist 既にFTPサーバ側にフォルダが作成されている時のエラー
var ErrFileExist = errors.New("550 Cannot create a file when that file already exists. ")

// FTPConfigListInfo FTP情報
type FTPConfigListInfo struct {
	// FTPサーバ側情報
	FTP           int             // FTP識別ナンバー
	conn          *ftp.ServerConn // コネクション（クライアント <-> FTPサーバ間）
	HostName      string          // FTPサーバホスト名
	User          string          // ログインID（IP)
	Password      string          // FTPログオンパスワード
	Folder        string          // 送信フォルダ
	FTPServerPort uint16          // FTPサーバポート
	// クライアント側情報
	Cycle         int    // 転送周期／リトライ周期
	Retry         int    // 転送リトライ数
	SendHour      string // 送信時刻（時）
	SendMin       string // 送信時刻（分）
	TransPath     string // 送信フォルダ
	MovePath      string // 移動フォルダ
	Ident         string // 対象ファイル識別
	TransFileName string // 送信ファイル名
	isOnTimeSend  int    // 1: 指定時刻送信【HH:MM】, 1以外: 定周期送信（Cycle）
}

func NewFTPConfigListInfo(FTPNumber int) *FTPConfigListInfo {
	cfg, err := ini.Load(iniFile)
	if err != nil {
		log.Fatalf("Tried to read %s but failed because %s", iniFile, err.Error())
	}

	sectionName := "FTP" + strconv.Itoa(FTPNumber)

	return &FTPConfigListInfo{
		// FTPサーバ情報
		FTP:           FTPNumber,
		conn:          nil,
		HostName:      cfg.Section(sectionName).Key("HostName").MustString("192.168.8.223"),
		User:          cfg.Section(sectionName).Key("User").MustString("Administrator"),
		Password:      cfg.Section(sectionName).Key("Password").MustString(""),
		Folder:        cfg.Section(sectionName).Key("Folder").MustString("sub2"),
		FTPServerPort: uint16(cfg.Section(sectionName).Key("Port").MustUint(21)),

		// クライアント情報
		Cycle:         cfg.Section(sectionName).Key("Cycle").MustInt(1),
		Retry:         cfg.Section(sectionName).Key("Retry").MustInt(5),
		SendHour:      cfg.Section(sectionName).Key("SendHour").MustString("12"),
		SendMin:       cfg.Section(sectionName).Key("SendMin").MustString("0"),
		TransPath:     cfg.Section(sectionName).Key("TransPath").MustString(""),
		MovePath:      cfg.Section(sectionName).Key("MovePath").MustString(""),
		Ident:         cfg.Section(sectionName).Key("Ident").MustString(""),
		TransFileName: cfg.Section(sectionName).Key("TransFileName").MustString(""),
		isOnTimeSend:  cfg.Section(sectionName).Key("isOnTimeSend").MustInt(1),
	}
}

func (f *FTPConfigListInfo) Run(wg *sync.WaitGroup) {
	defer wg.Done()

	// 定周期送信用
	t := time.NewTicker(time.Duration(f.Cycle))
	// 指定時刻用
	timeMatch := make(chan bool)

	for {
		now := fmt.Sprintf("%s", strconv.Itoa(time.Now().Hour())+":"+strconv.Itoa(time.Now().Minute()))
		// ？　現在時刻　HH：MM == 指定時刻
		if now == f.SendHour+":"+f.SendMin {
			timeMatch <- true
		}

		select {
		case <-t.C:
			// ? 定期送信
			if f.isOnTimeSend == 1 {
				f.FTPSendFile()
			}

			t.Reset(time.Duration(f.Cycle))
		case <-timeMatch:
			// ? 指定時刻
			if f.isOnTimeSend != 1 {
				f.FTPSendFile()
			}
		}
	}
}

// NotifyChangeIniSettings will be called by c#
// in order to reflect ini settings change.
//
//export NotifyChangeIniSettings
//func NotifyChangeIniSettings(FTPNumber int) {
//	app := NewFTPConfigListInfo(FTPNumber)
//	app.Run()
//}

func (f *FTPConfigListInfo) TryFTPLogin() error {
	// ？　ログイン成功
	retryCount := 0

	for i := 0; i < f.Retry; i++ {
		err := f.FTPLogin()
		if err != nil {
			retryCount++
			log.Printf("Failed to Login %s remote server %d times", f.HostName, retryCount)
			if retryCount == f.Retry {
				return err
			}

			// リトライ周期
			time.Sleep(time.Duration(f.Cycle))
		} else {
			break
		}
	}

	// 成功
	return nil
}

func (f *FTPConfigListInfo) FTPLogin() error {
	ftpAddr := f.HostName + ":" + strconv.Itoa(int(f.FTPServerPort))
	conn, err := ftp.Dial(ftpAddr)
	if err != nil {
		log.Printf("Tried to connect FTP Server at %s, but failed", ftpAddr)
		return err
	}

	err = conn.Login(f.User, f.Password)
	if err != nil {
		log.Printf("Tried to Login in %s by `%s` but failed", f.HostName, f.Password)
		return err
	}

	// コネクション（CL - SV）
	f.conn = conn

	return nil
}

func (f *FTPConfigListInfo) FTPRenameFile(newname string) error {
	return f.conn.Rename(f.TransPath, newname)
}

func (f *FTPConfigListInfo) getAllPath(target *[]string) {
	all, err := os.ReadDir(f.TransPath + "\\")
	if err != nil {
		return
	}

	var checkRegexp func(string) bool
	pattern := f.Ident

	checkRegexp = func(fileName string) bool {
		matched, err := regexp.MatchString(pattern, fileName)
		if err != nil || !matched {
			return false
		}

		return true
	}

	for _, file := range all {
		// ファイル名が合致 又は ファイル識別（正規表現）合致
		if file.Name() == f.TransFileName || checkRegexp(file.Name()) {
			*target = append(*target, file.Name())
		}
	}
}

func (f *FTPConfigListInfo) FTPSendFile() {
	// ログイン試行
	if err := f.TryFTPLogin(); err != nil {
		return
	}
	// 終了
	defer func() {
		err := f.FTPQuit()
		if err != nil {
			log.Printf("Tried to quit FTP application, but failed because `%s`", err.Error())
		}
	}()

	// ？　保存先フォルダ作成
	err := f.checkFTPServerFolder()
	if err != nil {
		log.Printf("Tried to make `%s` Folder, but failed because %s", f.HostName, err.Error())
		return
	}

	targetFiles := make([]string, 0)
	f.getAllPath(&targetFiles)

	// ？　送信する対象ファイルが不存在
	if len(targetFiles) == 0 {
		log.Printf("There is no target file in the `%s` directory", f.TransPath)
		return
	}

	// ？　ファイル読み込み成功
	for _, filename := range targetFiles {
		body, err := os.ReadFile(f.TransPath + "\\" + filename)
		if err != nil {
			log.Printf("Tried to read `%s` file and '%s` but failed, "+
				"because %s", f.TransFileName, f.Ident, err.Error())
			continue
		}

		FTPServerFolder := f.Folder + "\\" + filename
		//　？　送信成功
		err = f.conn.Stor(FTPServerFolder, bytes.NewBuffer(body))
		if err != nil {
			log.Printf("Tried to send `%s` File, but failed, because %s", f.TransFileName, err.Error())
		}
	}
}

func (f *FTPConfigListInfo) checkFTPServerFolder() error {
	err := f.conn.MakeDir(f.Folder)
	if err != nil {
		// ? 既にフォルダ作成済
		if err.Error() == ErrFileExist.Error() {
			log.Printf("Tried to make %s Folder, but already exists", f.Folder)
			return nil
		}
	}

	return err
}

func (f *FTPConfigListInfo) FTPQuit() error {
	return f.conn.Quit()
}

func (f *FTPConfigListInfo) WatchFTPServerFiles() {
	entries, err := f.conn.List(".")
	if err != nil {
		log.Printf("Tried to watch files inside FTP Server, but failed because `%s`", err.Error())
		return
	}

	for _, entry := range entries {
		switch entry.Type {
		case ftp.EntryTypeFile:
			fmt.Println("file -> " + entry.Name)
		case ftp.EntryTypeFolder:
			fmt.Println("folder -> " + entry.Name)
		case ftp.EntryTypeLink:
			fmt.Println("link ->" + entry.Name)
		default:
			fmt.Println("I do not know" + entry.Type.String())
		}
	}
}
