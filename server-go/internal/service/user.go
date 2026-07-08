package service

import (
	"errors"
	"time"

	"dalanshu/internal/db"
	"dalanshu/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户域业务：注册、登录、查询。
type UserService struct {
	db *db.DBSet
}

func NewUserService(d *db.DBSet) *UserService {
	return &UserService{db: d}
}

// Register 注册（写主库），返回新建用户。已存在用户名返回 ErrDuplicate。
func (s *UserService) Register(username, password, avatar string) (*model.User, error) {
	exists, err := s.Exists(username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicate
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}
	av := avatar
	if av == "" {
		av = "😎"
	}
	user := model.User{
		Username:     username,
		PasswordHash: string(hash),
		Avatar:       av,
		Bio:          "新来的散帅，请多关照 🌞",
		CreatedAt:    time.Now().UnixMilli(),
	}
	if err := s.db.W().Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Login 登录（读从/主库），校验通过返回用户，否则 ErrNotFound / ErrPassword。
func (s *UserService) Login(username, password string) (*model.User, error) {
	var user model.User
	if err := s.db.R().Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrPassword
	}
	return &user, nil
}

// Get 按 ID 取用户（匿名/未找到返回 nil）。
func (s *UserService) Get(uid int64) *model.User {
	if uid == 0 {
		return nil
	}
	var u model.User
	if err := s.db.R().First(&u, uid).Error; err != nil {
		return nil
	}
	return &u
}

// Exists 用户名是否已存在。
func (s *UserService) Exists(username string) (bool, error) {
	var n int64
	if err := s.db.R().Model(&model.User{}).Where("username = ?", username).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}


