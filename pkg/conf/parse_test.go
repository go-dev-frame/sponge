package conf

import (
	"fmt"
	"os"
	"testing"
	"time"
)

var c = make(map[string]interface{})

func TestShow(t *testing.T) {
	t.Log(Show(c))
	t.Log(Show(make(chan string)))
}

func Test_replaceDSN(t *testing.T) {
	dsn := "default:123456@192.168.3.37:6379/0"
	t.Log(replaceDSN(dsn))

	dsn = "default:123456:192.168.3.37:6379/0"
	t.Log(replaceDSN(dsn))
}

func Test_hideSensitiveFields(t *testing.T) {
	var keywords []string
	keywords = append(keywords, `"dsn"`, `"password"`, `"name"`)
	str := Show(c, keywords...)

	fmt.Printf(hideSensitiveFields(str))

	str = "\ndefault:123456@192.168.3.37:6379/0\n"
	fmt.Printf(hideSensitiveFields(str))
}

// test listening for configuration file updates
func TestParse(t *testing.T) {
	conf := make(map[string]interface{})

	handleChangeConfig := func() {
		fmt.Println("config file changed, do something here")
	}

	err := Parse("test.yml", &conf, handleChangeConfig)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Second)
	content, _ := os.ReadFile("test.yml")
	contentChange := append(content, byte('#'))
	time.Sleep(time.Millisecond * 100)
	_ = os.WriteFile("test.yml", contentChange, 0666) // change file
	time.Sleep(time.Millisecond * 100)
	_ = os.WriteFile("test.yml", content, 0666) // recovery documents
	time.Sleep(time.Millisecond * 100)
}

func TestParseErr(t *testing.T) {
	// result error test
	err := Parse("test.yml", nil)
	t.Log(err)

	// not found error test
	err = Parse("notfound.yml", &c)
	t.Log(err)
}

func TestParseConfigData(t *testing.T) {
	conf := make(map[string]interface{})

	data, err := os.ReadFile("test.yml")
	if err != nil {
		t.Error(err)
		return
	}
	err = ParseConfigData(data, "yaml", &conf)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(Show(conf))
}
