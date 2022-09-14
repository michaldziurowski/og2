package og2

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type rate struct {
	amount   int
	interval time.Duration
}

type upgradeInfo struct {
	running      bool
	whenFinished time.Time
}

type cost struct {
	ironAmount   int64
	copperAmount int64
	goldAmount   int64
}

type factoryLevel int

type FactoryInfo struct {
	Amount          int64
	Interval        time.Duration
	upgradeDuration time.Duration
	upgradeCost     cost
}

type factoryConfig map[factoryLevel]FactoryInfo

var ironFactoryConfig = factoryConfig{
	1: {
		Amount:          10,
		Interval:        time.Second,
		upgradeDuration: 15 * time.Second,
		upgradeCost: cost{
			ironAmount:   300,
			copperAmount: 100,
			goldAmount:   1,
		},
	},
	2: {
		Amount:          20,
		Interval:        time.Second,
		upgradeDuration: 30 * time.Second,
		upgradeCost: cost{
			ironAmount:   800,
			copperAmount: 250,
			goldAmount:   2,
		},
	},
	3: {
		Amount:          40,
		Interval:        time.Second,
		upgradeDuration: 60 * time.Second,
		upgradeCost: cost{
			ironAmount:   1600,
			copperAmount: 500,
			goldAmount:   4,
		},
	},
	4: {
		Amount:          80,
		Interval:        time.Second,
		upgradeDuration: 90 * time.Second,
		upgradeCost: cost{
			ironAmount:   3000,
			copperAmount: 1000,
			goldAmount:   8,
		},
	},
	5: {
		Amount:          150,
		Interval:        time.Second,
		upgradeDuration: 120 * time.Second,
		upgradeCost:     cost{},
	},
}

var copperFactoryConfig = factoryConfig{
	1: {
		Amount:          3,
		Interval:        time.Second,
		upgradeDuration: 15 * time.Second,
		upgradeCost: cost{
			ironAmount:   200,
			copperAmount: 70,
		},
	},
	2: {
		Amount:          7,
		Interval:        time.Second,
		upgradeDuration: 30 * time.Second,
		upgradeCost: cost{
			ironAmount:   400,
			copperAmount: 150,
		},
	},
	3: {
		Amount:          14,
		Interval:        time.Second,
		upgradeDuration: 60 * time.Second,
		upgradeCost: cost{
			ironAmount:   800,
			copperAmount: 300,
		},
	},
	4: {
		Amount:          30,
		Interval:        time.Second,
		upgradeDuration: 90 * time.Second,
		upgradeCost: cost{
			ironAmount:   1600,
			copperAmount: 600,
		},
	},
	5: {
		Amount:          60,
		Interval:        time.Second,
		upgradeDuration: 120 * time.Second,
		upgradeCost:     cost{},
	},
}

var goldFactoryConfig = factoryConfig{
	1: {
		Amount:          2,
		Interval:        time.Minute,
		upgradeDuration: 15 * time.Second,
		upgradeCost: cost{
			copperAmount: 100,
			goldAmount:   2,
		},
	},
	2: {
		Amount:          3,
		Interval:        time.Minute,
		upgradeDuration: 30 * time.Second,
		upgradeCost: cost{
			copperAmount: 200,
			goldAmount:   4,
		},
	},
	3: {
		Amount:          4,
		Interval:        time.Minute,
		upgradeDuration: 60 * time.Second,
		upgradeCost: cost{
			copperAmount: 400,
			goldAmount:   8,
		},
	},
	4: {
		Amount:          6,
		Interval:        time.Minute,
		upgradeDuration: 90 * time.Second,
		upgradeCost: cost{
			copperAmount: 800,
			goldAmount:   16,
		},
	},
	5: {
		Amount:          8,
		Interval:        time.Minute,
		upgradeDuration: 120 * time.Second,
		upgradeCost:     cost{},
	},
}

type FactoryType string

const (
	IRON_FACTORY   FactoryType = "iron"
	COPPER_FACTORY FactoryType = "copper"
	GOLD_FACTORY   FactoryType = "gold"
)

type Factory struct {
	Level             factoryLevel
	LevelConfig       FactoryInfo
	Config            factoryConfig
	UpgradeInProgress bool
	productionC       chan int64
	upgradeMu         sync.Mutex
	upgradeDoneC      chan bool
}

func newFactory(config factoryConfig) *Factory {
	return &Factory{
		Level:       1,
		LevelConfig: config[1],
		Config:      config,
		productionC: make(chan int64),
	}
}

func (f *Factory) start() {
	ticker := time.NewTicker(f.LevelConfig.Interval)
	for {
		<-ticker.C
		f.productionC <- f.LevelConfig.Amount
	}
}

func (f *Factory) canUpgrade(iron int64, copper int64, gold int64) bool {
	return !f.UpgradeInProgress && (iron > f.LevelConfig.upgradeCost.ironAmount && copper > f.LevelConfig.upgradeCost.copperAmount && gold > f.LevelConfig.upgradeCost.goldAmount)
}

func (f *Factory) upgrade() error {
	f.upgradeMu.Lock()
	defer f.upgradeMu.Unlock()
	if f.UpgradeInProgress {
		return errors.New("cannot upgrade factory, another upgrade in progress")
	}

	f.UpgradeInProgress = true
	go func() {
		time.Sleep(f.LevelConfig.upgradeDuration)
		f.upgradeDoneC <- true
	}()

	go func() {
		<-f.upgradeDoneC

		f.Level += 1
		f.LevelConfig = f.Config[f.Level]
		f.UpgradeInProgress = false
	}()

	return nil
}

type game struct {
	users map[string]*User
}

func NewGame() *game {
	return &game{
		users: map[string]*User{},
	}
}

func (g *game) RegisterUser(name string) {
	u := newUser(name)
	g.users[name] = u
	u.begin()
}

func (g *game) GetUser(name string) *User {
	return g.users[name]
}

func (g *game) UpgradeUserFactory(name string, fType FactoryType) error {
	u := g.GetUser(name)
	return u.upgrade(fType)
}

type Possesion struct {
	Amount  int64
	Factory *Factory
}

func newPossesion(f *Factory) *Possesion {
	return &Possesion{
		Amount:  0,
		Factory: f,
	}
}

type User struct {
	Username        string
	IronPossesion   *Possesion
	CopperPossesion *Possesion
	GoldPossesion   *Possesion
}

func newUser(name string) *User {
	return &User{
		Username:        name,
		IronPossesion:   newPossesion(newFactory(ironFactoryConfig)),
		CopperPossesion: newPossesion(newFactory(copperFactoryConfig)),
		GoldPossesion:   newPossesion(newFactory(goldFactoryConfig)),
	}
}

func (u *User) begin() {
	// TODO: handle goroutines termination
	go u.IronPossesion.Factory.start()
	go u.CopperPossesion.Factory.start()
	go u.GoldPossesion.Factory.start()

	go func() {
		for {
			procudedAmount := <-u.IronPossesion.Factory.productionC
			atomic.AddInt64(&u.IronPossesion.Amount, procudedAmount)
		}
	}()

	go func() {
		for {
			procudedAmount := <-u.CopperPossesion.Factory.productionC
			atomic.AddInt64(&u.CopperPossesion.Amount, procudedAmount)
		}
	}()

	go func() {
		for {
			procudedAmount := <-u.GoldPossesion.Factory.productionC
			atomic.AddInt64(&u.GoldPossesion.Amount, procudedAmount)
		}
	}()
}

func (u *User) upgrade(fType FactoryType) error {
	switch fType {
	case IRON_FACTORY:
		if u.IronPossesion.Factory.canUpgrade(u.IronPossesion.Amount, u.CopperPossesion.Amount, u.GoldPossesion.Amount) {
			atomic.AddInt64(&u.IronPossesion.Amount, -u.IronPossesion.Factory.LevelConfig.upgradeCost.ironAmount)
			atomic.AddInt64(&u.CopperPossesion.Amount, -u.CopperPossesion.Factory.LevelConfig.upgradeCost.copperAmount)
			atomic.AddInt64(&u.GoldPossesion.Amount, -u.GoldPossesion.Factory.LevelConfig.upgradeCost.goldAmount)

			err := u.IronPossesion.Factory.upgrade()
			if err != nil {
				// compensate for substraction
				atomic.AddInt64(&u.IronPossesion.Amount, u.IronPossesion.Factory.LevelConfig.upgradeCost.ironAmount)
				atomic.AddInt64(&u.CopperPossesion.Amount, u.CopperPossesion.Factory.LevelConfig.upgradeCost.copperAmount)
				atomic.AddInt64(&u.GoldPossesion.Amount, u.GoldPossesion.Factory.LevelConfig.upgradeCost.goldAmount)
			}

		} else {
			return fmt.Errorf("cannot upgrade factory %s", fType)
		}
	case COPPER_FACTORY:
		if u.CopperPossesion.Factory.canUpgrade(u.IronPossesion.Amount, u.CopperPossesion.Amount, u.GoldPossesion.Amount) {
			atomic.AddInt64(&u.IronPossesion.Amount, -u.IronPossesion.Factory.LevelConfig.upgradeCost.ironAmount)
			atomic.AddInt64(&u.CopperPossesion.Amount, -u.CopperPossesion.Factory.LevelConfig.upgradeCost.copperAmount)
			atomic.AddInt64(&u.GoldPossesion.Amount, -u.GoldPossesion.Factory.LevelConfig.upgradeCost.goldAmount)

			err := u.CopperPossesion.Factory.upgrade()
			if err != nil {
				// compensate for substraction
				atomic.AddInt64(&u.IronPossesion.Amount, u.IronPossesion.Factory.LevelConfig.upgradeCost.ironAmount)
				atomic.AddInt64(&u.CopperPossesion.Amount, u.CopperPossesion.Factory.LevelConfig.upgradeCost.copperAmount)
				atomic.AddInt64(&u.GoldPossesion.Amount, u.GoldPossesion.Factory.LevelConfig.upgradeCost.goldAmount)
			}

		} else {
			return fmt.Errorf("cannot upgrade factory %s", fType)
		}
	case GOLD_FACTORY:
		if u.GoldPossesion.Factory.canUpgrade(u.IronPossesion.Amount, u.CopperPossesion.Amount, u.GoldPossesion.Amount) {
			atomic.AddInt64(&u.IronPossesion.Amount, -u.IronPossesion.Factory.LevelConfig.upgradeCost.ironAmount)
			atomic.AddInt64(&u.CopperPossesion.Amount, -u.CopperPossesion.Factory.LevelConfig.upgradeCost.copperAmount)
			atomic.AddInt64(&u.GoldPossesion.Amount, -u.GoldPossesion.Factory.LevelConfig.upgradeCost.goldAmount)

			err := u.GoldPossesion.Factory.upgrade()
			if err != nil {
				// compensate for substraction
				atomic.AddInt64(&u.IronPossesion.Amount, u.IronPossesion.Factory.LevelConfig.upgradeCost.ironAmount)
				atomic.AddInt64(&u.CopperPossesion.Amount, u.CopperPossesion.Factory.LevelConfig.upgradeCost.copperAmount)
				atomic.AddInt64(&u.GoldPossesion.Amount, u.GoldPossesion.Factory.LevelConfig.upgradeCost.goldAmount)
			}

		} else {
			return fmt.Errorf("cannot upgrade factory %s", fType)
		}
	}

	return nil
}
