package api

type ReloadMaterialsDto struct {
	ReloadMaterial []ReloadMaterialDto `json:"reloadMaterial"`
}

type ReloadMaterialDto struct {
	AppId         int    `json:"appId"`
	GitmaterialId int    `json:"gitMaterialId"`
	CloningMode   string `json:"cloningMode"`
}
