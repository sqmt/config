package config

import (
    "bytes"
    "fmt"
    "os"
    "path"
    "strings"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/spf13/viper"
    _ "github.com/spf13/viper/remote"
)

var (
    // 配置文件的系统环境变量名，例如：MY_CONFIG_FILE，优先级 小于 configFile
    envFileKey string
    // 配置名称， 默认是 config ，结合 configType 和 searchPath 组成完整路径，优先级最低
    configName string
    // 配置类型, 默认是 yaml
    configType string
    // 指定配置文件路径，例如：./config/config.yaml, 如果有，优先级最高
    configFile string
    // 配置文件搜索目录，在指定的一个或多个目录中按照目录传入顺序查找 configName 和 configType
    searchPath []string
    // 默认环境变量的前缀，例如: MY_
    envPrefix string
    // 当配置发生变化时 需要执行的操作
    watchHandler func(v *viper.Viper)
)

// SetEnvFileKey 设置配置文件的系统环境变量名
func SetEnvFileKey(val string) {
    if envFileKey == "" {
        envFileKey = val
    }
}

// SetConfigName 设置配置名称
func SetConfigName(name string) {
    if configName == "" {
        configName = name
    }
}

// SetConfigType 设置配置类型
func SetConfigType(typ string) {
    if configType == "" {
        configType = typ
    }
}

// SetConfigFile 设置配置文件路径
func SetConfigFile(filename string) {
    if configFile == "" {
        configFile = filename
    }
}

// SetSearchPath 设置配置文件检索目录
func SetSearchPath(paths []string) {
    if len(searchPath) == 0 {
        searchPath = paths
    }
}

// SetEnvPrefix 设置默认环境变量的前缀
func SetEnvPrefix(prefix string) {
    if envPrefix == "" {
        envPrefix = prefix
    }
}

// SetWatchHandler 设置默认配置变动后执行的操作
func SetWatchHandler(handler func(v *viper.Viper)) {
    if watchHandler == nil {
        watchHandler = handler
    }
}

type Option struct {
    // 提供者，file, content，etcd, consul, firestore
    Provider       string   `json:"provider" yaml:"provider"`
    Name           string   `json:"name" yaml:"name"`
    Type           string   `json:"type" yaml:"type"`
    File           string   `json:"file" yaml:"file"`
    SearchPath     []string `json:"searchPath" yaml:"searchPath"`
    Env            bool     `json:"env" yaml:"env"`
    EnvPrefix      string   `json:"envPrefix" yaml:"envPrefix"`
    EndPoint       string   `json:"endPoint" yaml:"endPoint"`
    SecretKey      string   `json:"secretKey" yaml:"secretKey"`
    Watch          bool     `json:"watch" yaml:"watch"`
    IsRemote       bool     `json:"isRemote" yaml:"isRemote"`
    RemoteProvider string   `json:"remoteProvider" yaml:"remoteProvider"`
    RemoteEndpoint string   `json:"remote_endpoint" yaml:"remote_endpoint"`
    WatchHandler   func(v *viper.Viper)
}

// getConfigType 获取配置类型
func (o *Option) getConfigType() string {
    if o.Type != "" {
        return o.Type
    } else if configType != "" {
        return configType
    } else {
        return "yaml"
    }
}

// getConfigName 获取配置名称
func (o *Option) getConfigName() string {
    if o.Name != "" {
        return o.Name
    } else if configName != "" {
        return configName
    } else {
        return "config"
    }
}

// getConfigFile 获取配置文件
func (o *Option) getConfigFile() string {
    if o.File != "" {
        return o.File
    }

    if envFileKey != "" && os.Getenv(envFileKey) != "" {
        return os.Getenv(envFileKey)
    }

    return ""
}

// getSearchPath 获取配置文件检索目录
func (o *Option) getSearchPath() []string {
    if len(o.SearchPath) > 0 {
        return o.SearchPath
    } else if len(searchPath) > 0 {
        return searchPath
    }
    pwd, _ := os.Getwd()

    return []string{pwd, path.Join(pwd, "config")}
}

// getWatchHandler 获取默认配置变动后执行的操作
func (o *Option) getWatchHandler() func(v *viper.Viper) {
    if o.WatchHandler != nil {
        return o.WatchHandler
    }

    return watchHandler
}

// New returns a new configuration management object.
func New(option ...*Option) (v *viper.Viper, err error) {
    o := &Option{}
    if len(option) > 0 && option[0] != nil {
        o = option[0]
    }
    v = viper.New()
    switch strings.ToLower(o.Provider) {
    case "file":
        err = fileProvider(v, o)
    case "content":
        err = contentProvider(v, o)
    case "etcd", "consul", "firestore":
        err = remoteProvider(v, o)
    default:
        err = fileProvider(v, o)
    }

    if err != nil {
        return v, err
    }
    env(v, o)
    return v, nil
}

// fileProvider 文件提供者
func fileProvider(v *viper.Viper, o *Option) error {
    if f := o.getConfigFile(); f != "" {
        v.SetConfigFile(f)
    } else {
        for _, s := range o.getSearchPath() {
            v.AddConfigPath(s)
        }
        v.SetConfigName(o.getConfigName())
        v.SetConfigType(o.getConfigType())
    }
    if err := v.ReadInConfig(); err != nil {
        return err
    }
    if o.Watch {
        fileWatch(v, o)
    }

    return nil
}

// remoteProvider 读取远程配置
func remoteProvider(v *viper.Viper, o *Option) error {
    var err error
    if o.SecretKey != "" {
        err = v.AddSecureRemoteProvider(o.Provider, o.EndPoint, o.getConfigFile(), o.SecretKey)
    } else {
        err = v.AddRemoteProvider(o.Provider, o.EndPoint, o.getConfigFile())
    }
    if err != nil {
        return err
    }
    v.SetConfigType(o.getConfigType())
    if err = v.ReadRemoteConfig(); err != nil {
        return err
    }
    if o.Watch {
        remoteWatch(v, o)
    }

    return nil
}

// contentProvider 文本内容提供者
func contentProvider(v *viper.Viper, o *Option) error {
    b := bytes.NewBufferString(o.File)
    v.SetConfigType(o.getConfigType())

    return v.MergeConfig(b)
}

// fileWatch 文件变动
func fileWatch(v *viper.Viper, o *Option) {
    v.WatchConfig()
    if fn := o.getWatchHandler(); fn != nil {
        v.OnConfigChange(func(in fsnotify.Event) {
            fn(v)
        })
    }
}

// remoteWatch 远程文件变动
func remoteWatch(v *viper.Viper, o *Option) {
    fn := o.getWatchHandler()

    go func() {
        for {
            time.Sleep(time.Second * 5) // delay after each request

            // currently, only tested with etcd support
            err := v.WatchRemoteConfig()
            if err != nil {
                fmt.Println("unable to read remote config:", err)
                continue
            }
            if fn != nil {
                fn(v)
            }
        }
    }()
}

// env 处理env
func env(v *viper.Viper, o *Option) {
    if !o.Env {
        return
    }
    v.AutomaticEnv()
    v.SetEnvPrefix(o.EnvPrefix)
}
