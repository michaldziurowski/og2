package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/michaldziurowski/og2"
)

type CreateUserCmd struct {
	UserName string `json:"username"`
}
type UpgradeUserFactoryCmd struct {
	UserName    string `json:"username"`
	FactoryType string `json:"factoryType"`
}

type ProductionRateInfo struct {
	Amount   int64         `json:"amount"`
	Interval string `json:"inteval"`
}

type UpgradeInfo struct {
	InProgress bool `json:"inprogress"`
}

type FactoryInfo struct {
	Level          int64              `json:"level"`
	ProductionRate ProductionRateInfo `json:"productionRate"`
	Upgrade        UpgradeInfo        `json:"upgrade"`
}

type UserPossessionsInfo struct {
	Amount  int64       `json:"amount"`
	Factory FactoryInfo `json:"factory"`
}

func NewUserPossesionInfo(userPossesion *og2.Possesion) UserPossessionsInfo {
	return UserPossessionsInfo{
		Amount: userPossesion.Amount,
		Factory: FactoryInfo{
            Level: int64(userPossesion.Factory.Level),
			ProductionRate: ProductionRateInfo{
				Amount:   userPossesion.Factory.LevelConfig.Amount,
				Interval: userPossesion.Factory.LevelConfig.Interval.String(),
			},
			Upgrade: UpgradeInfo{
				InProgress: userPossesion.Factory.UpgradeInProgress,
			},
		},
	}
}

type UserInfo struct {
	UserName         string              `json:"username"`
	IronPossession   UserPossessionsInfo `json:"ironPossesion"`
	CopperPossession UserPossessionsInfo `json:"copperPossesion"`
	GoldPossession   UserPossessionsInfo `json:"goldPossesion"`
}


func main() {
	g := og2.NewGame()
	e := echo.New()
	e.POST("/user", func(c echo.Context) error {
		cmd := new(CreateUserCmd)
		if err := c.Bind(cmd); err != nil {
			return err
		}

		g.RegisterUser(cmd.UserName)

		return c.JSON(http.StatusCreated, cmd)
	})

	e.GET("/dashboard/:username", func(c echo.Context) error {
		username := c.Param("username")

		// handle error when there is no user
		u := g.GetUser(username)

		r := UserInfo{
			UserName:         username,
			IronPossession:   NewUserPossesionInfo(u.IronPossesion),
			CopperPossession: NewUserPossesionInfo(u.CopperPossesion),
			GoldPossession:   NewUserPossesionInfo(u.GoldPossesion),
		}
		return c.JSON(http.StatusOK, r)
	})

	e.POST("/upgrade", func(c echo.Context) error {
		cmd := new(UpgradeUserFactoryCmd)
		if err := c.Bind(cmd); err != nil {
			return err
		}

		// should validate if valid factory type
		err := g.UpgradeUserFactory(cmd.UserName, og2.FactoryType(cmd.FactoryType))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

        // should return user status after upgrade not cmd
		return c.JSON(http.StatusOK, cmd)
	})

	e.Logger.Fatal(e.Start(":1323"))
}

