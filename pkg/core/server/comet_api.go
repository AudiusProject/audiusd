// module that exposes cometbft rpc endpoints to support state syncing.
package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (s *Server) registerCometRPC() {
	e := s.GetEcho()

	g := e.Group("/core/crpc")

	g.GET("/status", s.handleCometRPCStatus)
	g.GET("/block", s.handleCometRPCBlock)
	g.GET("/commit", s.handleCometRPCCommit)
	g.GET("/validators", s.handleCometRPCValidators)
	g.GET("/genesis", s.handleCometRPCGenesis)
}

func (s *Server) checkCometRPC() error {
	if s.rpc == nil {
		return fmt.Errorf("comet rpc is not enabled")
	}
	return nil
}

func (s *Server) handleCometRPCStatus(c echo.Context) error {
	if err := s.checkCometRPC(); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	status, err := s.rpc.Status(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, status)
}

func (s *Server) handleCometRPCBlock(c echo.Context) error {
	if err := s.checkCometRPC(); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	var heightPtr *int64
	if heightStr := c.QueryParam("height"); heightStr != "" {
		h, err := strconv.ParseInt(heightStr, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "invalid height")
		}
		heightPtr = &h
	}

	block, err := s.rpc.Block(c.Request().Context(), heightPtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, block)
}

func (s *Server) handleCometRPCCommit(c echo.Context) error {
	if err := s.checkCometRPC(); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	var heightPtr *int64
	if heightStr := c.QueryParam("height"); heightStr != "" {
		h, err := strconv.ParseInt(heightStr, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "invalid height")
		}
		heightPtr = &h
	}

	commit, err := s.rpc.Commit(c.Request().Context(), heightPtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, commit)
}

func (s *Server) handleCometRPCValidators(c echo.Context) error {
	if err := s.checkCometRPC(); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	var heightPtr *int64
	if heightStr := c.QueryParam("height"); heightStr != "" {
		h, err := strconv.ParseInt(heightStr, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "invalid height")
		}
		heightPtr = &h
	}

	var pagePtr *int
	if pageStr := c.QueryParam("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "invalid page")
		}
		pagePtr = &p
	}

	var perPagePtr *int
	if ppStr := c.QueryParam("per_page"); ppStr != "" {
		pp, err := strconv.Atoi(ppStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, "invalid per_page")
		}
		perPagePtr = &pp
	}

	validators, err := s.rpc.Validators(c.Request().Context(), heightPtr, pagePtr, perPagePtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, validators)
}

func (s *Server) handleCometRPCGenesis(c echo.Context) error {
	if err := s.checkCometRPC(); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	genesis, err := s.rpc.Genesis(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, genesis)
}
