package config

import (
    "os"
    "os/exec"
    "path"
    "testing"
    "time"
)

var (
    testEnvFileKey         = "MY_TEST_CONFIG_FILE"
    testFileOptionWithName = &Option{Provider: "file", Name: "test"}
    testFileOptionWithType = &Option{Provider: "file", Name: "test1", Type: "toml"}
    testRemoteOption       = &Option{Provider: "etcd", File: "/test.toml", EndPoint: "http://127.0.0.1:2379", Type: "toml"}
    testContentOption      = &Option{
        Provider: "content",
        File: string([]byte(`
title = "TOML Example"

[owner]
organization = "MongoDB"
Bio = "MongoDB Chief Developer Advocate & Hacker at Large"
dob = 1979-05-27T07:32:00Z`)),
        Type: "toml",
    }
)

func reset() {
    if envFileKey != "" {
        _ = os.Setenv(envFileKey, "")
    }
    envFileKey = ""
    configName = ""
    configType = ""
    configFile = ""
    searchPath = []string{}
    envPrefix = ""
    watchHandler = nil
}

func TestNew_Empty(t *testing.T) {
    defer reset()
    if _, err := New(); err == nil {
        t.Errorf("默认获取yaml的文件不存在, 期望报错")
    }
    SetConfigName("test")
    if v, err := New(); err != nil {
        t.Errorf("配置文件test.yaml存在,但是报错，err: %v", err)
    } else if u := v.ConfigFileUsed(); path.Base(u) != "test.yaml" {
        t.Errorf("期望读取的配置文件是test.yaml, 但是获取到的是 %v", u)
    }
}

func TestNew_Content(t *testing.T) {
    defer reset()
    if v, err := New(testContentOption); err != nil {
        t.Errorf("ContentProvider实例化失败，err: %v", err)
    } else if v.GetString("title") != "TOML Example" {
        t.Errorf("ContentProvider实例化成功，但是配置读取失败，期望 %v，获取: %v", "TOML Example", v.GetString("title"))
    }
}

func TestNew_FileEnv(t *testing.T) {
    defer reset()
    if err := os.Setenv(testEnvFileKey, "./test.yaml"); err != nil {
        t.Errorf("环境变量设置失败， err: %v", err)
    }
    SetEnvFileKey(testEnvFileKey)
    if v, err := New(); err != nil {
        t.Errorf("环境变量读取失败err: %v", err)
    } else if u := v.ConfigFileUsed(); path.Base(u) != "test.yaml" {
        t.Errorf("期望读取的配置文件是test.yaml, 但是获取到的是 %v", u)
    } else if !v.GetBool("Hacker") {
        t.Errorf("fileProvider实例化成功，但是配置读取失败，期望 %v，获取: %v", true, v.GetBool("Hacker"))
    }
}

func TestNew_Filename(t *testing.T) {
    defer reset()
    if v, err := New(testFileOptionWithName); err != nil {
        t.Errorf("实例化失败err: %v", err)
    } else if u := v.ConfigFileUsed(); path.Base(u) != "test.yaml" {
        t.Errorf("期望读取的配置文件是test.yaml, 但是获取到的是 %v", u)
    } else if !v.GetBool("Hacker") {
        t.Errorf("fileProvider实例化成功，但是配置读取失败，期望 %v，获取: %v", true, v.GetBool("Hacker"))
    }
}

func TestNew_FileType(t *testing.T) {
    defer reset()
    if v, err := New(testFileOptionWithType); err != nil {
        t.Errorf("实例化失败err: %v", err)
    } else if u := v.ConfigFileUsed(); path.Base(u) != "test1.toml" {
        t.Errorf("期望读取的配置文件是test1.toml, 但是获取到的是 %v", u)
    } else if v.GetString("title") != "TOML Example" {
        t.Errorf("ContentProvider实例化成功，但是配置读取失败，期望 %v，获取: %v", "TOML Example", v.GetString("title"))
    }
}

func TestNew_SearchPath(t *testing.T) {
    defer reset()
    SetSearchPath([]string{"./testdata"})
    if v, err := New(); err != nil {
        t.Errorf("配置文件./testdata/yaml存在,但是报错，err: %v", err)
    } else if u := v.ConfigFileUsed(); path.Base(u) != "config.yaml" {
        t.Errorf("期望读取的配置文件是config.yaml, 但是获取到的是 %v", u)
    } else if v.GetString("demo") != "test" {
        t.Errorf("实例化成功，但是配置读取失败，期望 %v，获取: %v", "test", v.GetString("demo"))
    }
}

func TestNew_Remote(t *testing.T) {
    defer reset()
    _initEtcd()
    if v, err := New(testRemoteOption); err != nil {
        t.Errorf("远程etcd配置实例化失败,err: %v", err)
    } else if v.GetString("demo") != "123" {
        t.Errorf("实例化成功，但是配置读取失败，期望 %v，获取: %v", "123", v.GetString("demo"))
    } else if v.GetString("test.title") != "TestTitle" {
        t.Errorf("实例化成功，但是配置读取失败，期望 %v，获取: %v", "TestTitle", v.GetString("test.title"))
    }
}

func _initEtcd() {
    etcdBin := os.Getenv("MY_TEST_ETCD")
    etcdctlBin := os.Getenv("MY_TEST_ETCDCTL")
    go func() {
        if err := exec.Command(etcdBin, "--data-dir", "./testdata/default.etcd").Run(); err != nil {
            panic(err)
        }
    }()
    time.Sleep(2 * time.Second)
    err := exec.Command(etcdctlBin, "set", "/test.toml", `
demo = "123"
[test]
title = "TestTitle"
`).Run()
    if err != nil {
        panic(err)
    }
}