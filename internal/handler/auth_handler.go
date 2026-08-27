package handler

import (
	"errors"
	"net/http"

	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{"Title": "登录"})
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{"Title": "注册"})
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	_, token, err := h.auth.Login(c.PostForm("email"), c.PostForm("password"))
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Title": "登录", "Error": err.Error()})
		return
	}
	setTokenCookie(c, token)
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) RegisterPage(c *gin.Context) {
	_, token, err := h.auth.Register(c.PostForm("name"), c.PostForm("email"), c.PostForm("password"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"Title": "注册", "Error": err.Error()})
		return
	}
	setTokenCookie(c, token)
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) LoginAPI(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "请求格式不正确"})
		return
	}
	user, token, err := h.auth.Login(input.Email, input.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) RegisterAPI(c *gin.Context) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "请求格式不正确"})
		return
	}
	user, token, err := h.auth.Register(input.Name, input.Email, input.Password)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrEmailExists) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("plant_diary_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

func setTokenCookie(c *gin.Context, token string) {
	c.SetCookie("plant_diary_token", token, 86400, "/", "", false, true)
}
