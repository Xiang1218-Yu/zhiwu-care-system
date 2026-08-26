package service

import (
	"testing"

	"plant-diary/api/dto"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/pkg/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestService 用内存 SQLite 搭建一套真实的 repository/service 栈，
// 用于验证浇水/施肥提交时记录与待办是否同步落库。
func newTestService(t *testing.T) (*PlantService, *model.Plant) {
	t.Helper()
	// 每个测试用独立命名的内存库，互不污染。
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	svc := NewPlantService(plants, care)

	// plants.user_id 外键指向 users，需先建用户。
	if err := db.Create(&model.User{ID: "user-1", Email: "u@example.com", Name: "测试"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	plant := &model.Plant{
		ID: "plant-1", UserID: "user-1", Name: "绿萝", Species: "Epipremnum",
		Source: "market", AcquiredDate: utils.Today(), Location: "balcony",
		Status: model.StatusHealthy, Difficulty: "easy",
	}
	if err := plants.Create(plant); err != nil {
		t.Fatalf("create plant: %v", err)
	}
	return svc, plant
}

// 浇水周期为 0 时必须拒绝：否则记录入库却无对应待办，造成历史与首页分叉。
func TestAddCareRejectsZeroCycle(t *testing.T) {
	svc, plant := newTestService(t)

	err := svc.AddCare("user-1", plant.ID, dto.CareInput{
		Type:       model.CareWater,
		WaterCycle: 0, // 即表单留空时 parseInt 得到的值
	}, "")
	if err == nil {
		t.Fatal("浇水周期为 0 时应被拒绝，避免记录与待办分叉")
	}

	// 校验阶段就应失败：既不应写入养护记录，也不应生成待办周期。
	logs, _ := svc.care.ListLogs(plant.ID)
	if len(logs) != 0 {
		t.Fatalf("周期非法时不应落库养护记录，实际写入 %d 条", len(logs))
	}
	cycles, _ := svc.care.ListDueCycles("user-1", utils.Today())
	if len(cycles) != 0 {
		t.Fatalf("周期非法时不应生成待办，实际生成 %d 条", len(cycles))
	}
}

// 施肥周期为空/越界同样应被拒绝，避免同一分叉问题。
func TestAddCareRejectsOutOfRangeCycle(t *testing.T) {
	svc, plant := newTestService(t)

	cases := []int{0, 366, -1}
	for _, days := range cases {
		err := svc.AddCare("user-1", plant.ID, dto.CareInput{
			Type:            model.CareFertilizer,
			FertilizerCycle: days,
		}, "")
		if err == nil {
			t.Fatalf("施肥周期 %d 应被拒绝", days)
		}
	}
	logs, _ := svc.care.ListLogs(plant.ID)
	if len(logs) != 0 {
		t.Fatalf("全部非法周期都不应落库记录，实际写入 %d 条", len(logs))
	}
}

// 合法周期时，养护记录与待办必须在同一事务内一并落库。
func TestAddCareStoresLogAndCycleTogether(t *testing.T) {
	svc, plant := newTestService(t)

	err := svc.AddCare("user-1", plant.ID, dto.CareInput{
		Type:       model.CareWater,
		WaterCycle: 3,
	}, "")
	if err != nil {
		t.Fatalf("合法周期应入库：%v", err)
	}

	logs, _ := svc.care.ListLogs(plant.ID)
	if len(logs) != 1 {
		t.Fatalf("应写入 1 条养护记录，实际 %d 条", len(logs))
	}
	cycles, _ := svc.care.ListDueCycles("user-1", utils.Today())
	if len(cycles) != 0 {
		// 周期 3 天，next_date 在未来，今日不应到期，但记录必须存在。
		t.Fatalf("3 天周期今日不应到期，待办数 %d", len(cycles))
	}
	// 直接查周期表，确认 next_date 已按周期推进到 3 天后。
	loaded, err := svc.Get("user-1", plant.ID)
	if err != nil {
		t.Fatalf("重新读取植物失败：%v", err)
	}
	cycle := loaded.Cycle(model.CareWater)
	if cycle == nil {
		t.Fatalf("待办周期应已落库")
	}
	want := utils.Today().AddDate(0, 0, 3)
	if !cycle.NextDate.Equal(want) {
		t.Fatalf("next_date 应为 %v，实际 %v", want, cycle.NextDate)
	}
}

// 换盆/修剪等无需周期的操作不应受影响。
func TestAddCareAllowsNonCyclicTypes(t *testing.T) {
	svc, plant := newTestService(t)

	for _, ct := range []string{model.CareRepot, model.CarePrune, model.CareSpray, model.CareClean} {
		if err := svc.AddCare("user-1", plant.ID, dto.CareInput{Type: ct}, ""); err != nil {
			t.Fatalf("无周期操作 %s 不应失败：%v", ct, err)
		}
	}
	logs, _ := svc.care.ListLogs(plant.ID)
	if len(logs) != 4 {
		t.Fatalf("应写入 4 条无周期养护记录，实际 %d 条", len(logs))
	}
}
