package api

type ReloadMaterialsDto struct {
	reloadMaterial []ReloadMaterialDto
}

type ReloadMaterialDto struct {
	AppId         int    `json:"appId"`
	GitmaterialId int    `json:"gitmaterialId"`
	CloningMode   string `json:"cloningMode"`
}
