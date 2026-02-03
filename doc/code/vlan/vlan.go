package netManage

import (
	"net"
	"netvine.com/firewall/server/utils/network"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"
	"netvine.com/firewall/server/global"
	"netvine.com/firewall/server/model/common/request"
	"netvine.com/firewall/server/model/netManage"
	"netvine.com/firewall/server/model/netManage/constant"
	"netvine.com/firewall/server/model/resourceObj"
)

type VlanConfigService struct {
}

func (s VlanConfigService) InfoUpdate(req netManage.VlanConfig) error {
	vlanMap := make(map[string]resourceObj.NetworkInterface)
	for _, value := range global.NETVINE_NICCONFIG {
		vlanMap[value.Name] = value
	}

	if constant.ROUTE_MODE == req.Type { // 路由模式
		var InterfaceConfigNetArray []netManage.InterfaceConfigNet
		copier.Copy(&InterfaceConfigNetArray, req.VlanConfigArray)
		err := checkAddress(InterfaceConfigNetArray)
		if err != nil {
			return err
		}
		if err == nil {
			var ipList []string
			err = global.NETVINE_DB.Transaction(func(tx *gorm.DB) error {
				err1 := global.NETVINE_DB.Unscoped().Delete(&[]netManage.VlanConfigArray{}, "out_interface = ?", req.OutInterface).Error
				if err1 != nil {
					return err1
				}
				//先清空
				err1 = network.VlanIfIpFlush(req.OutInterface)
				if err1 != nil {
					return err1
				}
				for _, v := range req.VlanConfigArray {
					if !strings.Contains(v.SubnetMask, ".") {
						atoi, _ := strconv.Atoi(v.SubnetMask)
						m := net.CIDRMask(atoi, 32)
						v.SubnetMask = net.IPv4(m[0], m[1], m[2], m[3]).String()
					}

					err1 = tx.Create(&netManage.VlanConfigArray{
						IpAddress:    v.IpAddress,
						SubnetMask:   v.SubnetMask,
						OutInterface: req.OutInterface,
					}).Error
					if err1 != nil {
						break
					}
					ipList = append(ipList, v.IpAddress+"/"+v.SubnetMask)
				}

				return err1
			})
			if err == nil {
				//处理命令
				//再新增
				err = network.VlanIfIpAdd(ipList, req.OutInterface)
				if err != nil {
					return err
				}
				global.NETVINE_DB.Model(&netManage.VlanConfig{}).Where("out_interface = ?", req.OutInterface).Select("nick_name", "type", "status").Updates(&req)
			}
		}
	} else {
		//清空
		err1 := network.VlanIfIpFlush(req.OutInterface)
		if err1 != nil {
			return err1
		}
		//更新接口模式类型
		req.Type = constant.TRANSPARENT_MODE
		global.NETVINE_DB.Model(&netManage.VlanConfig{}).Where("out_interface = ?", req.OutInterface).Select("nick_name", "type", "status").Updates(&req)
	}

	//网卡启停
	err2 := network.VlanIfUpAndDown(req.OutInterface, netManage.StatusMap[req.Status])
	return err2
}

func (s VlanConfigService) PageQuery(req request.PageStruct) (list []netManage.VlanConfigRes, total int64, err error) {
	limit := req.PageSize
	offset := req.PageSize * (req.Page - 1)

	var vlanConfigList []netManage.VlanConfig
	db := global.NETVINE_DB.Model(&netManage.VlanConfig{}).Preload("VlanConfigArray").Where("is_manager = ?", 0)
	err = db.Count(&total).Error
	err = db.Limit(limit).Offset(offset).Order("vlan_id asc").Offset(offset).Limit(limit).Find(&vlanConfigList).Error

	interfaceMap := make(map[string]string)
	for _, value := range global.NETVINE_NICCONFIG {
		interfaceMap[value.Name] = value.Id
	}

	for _, value := range vlanConfigList {
		var addressArray []string
		for _, v := range value.VlanConfigArray {
			addressArray = append(addressArray, v.IpAddress+"/"+v.SubnetMask)
		}
		list = append(list, netManage.VlanConfigRes{
			OutInterface:      value.OutInterface,
			Address:           strings.Join(addressArray, ","),
			NickName:          value.NickName,
			Type:              value.Type,
			PhysicalInterface: value.PhysicalInterface,
			VlanId:            value.VlanId,
			Status:            value.Status,
		})
	}
	return list, total, err
}

func (s VlanConfigService) DetailQuery(outInterface string) (list netManage.VlanConfig, err error) {
	err = global.NETVINE_DB.Model(&netManage.VlanConfig{}).Preload("VlanConfigArray").Where("out_interface = ?", outInterface).Find(&list).Error

	return list, err
}
