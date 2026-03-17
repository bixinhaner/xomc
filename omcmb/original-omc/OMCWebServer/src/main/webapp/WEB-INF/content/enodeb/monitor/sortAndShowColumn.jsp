<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.col-group {
		margin: 5px 10px;
		display: flex;
		flex-wrap: wrap;
	}
	.col-group .el-checkbox {
		min-width: 140px;
	}
	.col-group .el-checkbox__label {
		font-size: 12px;
	}

	.showHideItem input{
		margin-top:-2px;
		margin-bottom:1px;
		vertical-align:middle;
		margin-right:20px;
	}
	.selectAll{
		height:32px;
		width:334px;
		padding:28px 0px 0px 30px;
	}
	
	.showHideItem .select-all-cls {
		padding: 10px 0 0 9px;
		display: flex;
		align-items: center;
	}
	.select-all-cls > i {
		margin-right: 5px;
	}
	.showHideItem .select-all-cls > span {
		font-size: 14px;
		font-weight: bold;
		margin-left: 10px;
	}
	.showHideItem .el-icon-close1::before {
		color: #333;
	}
</style>
	<div class="flex-ctn" id="sortAndShowColumnBoxCls" style="height:540px;flex-direction: row;border-bottom: 1px solid #e9e9e9;">
		<div style="padding: 0px 0 0 10px;height:100%;flex:3;overflow:auto;">
			<div class="select-all-cls">
				<span style="margin: 0;font-weight: bold;"><%=rb.getString("XuanZeLie")%></span>
			</div>
			<div class="select-all-cls" style="padding-left: 10px;">
				<el-checkbox :indeterminate="!colAll" v-model="colAll" @change="colAllChange"></el-checkbox> 
				<span><%=rb.getString("QuanXuan")%></span>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.device,'el-icon-open':!expanded.device}" @click="expanded.device = !expanded.device"></i>
					<el-checkbox :indeterminate="form.device.length<deviceCol.length" v-model="deviceAll" @change="deviceAllChange"></el-checkbox> 
					<span><%=rb.getString("SheBeiXinXi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.device" class="col-group" v-model="form.device">
					<el-checkbox v-for="item in deviceCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.cell,'el-icon-open':!expanded.cell}" @click="expanded.cell = !expanded.cell"></i>
					<el-checkbox :indeterminate="form.cell.length<cellCol.length" v-model="cellAll" @change="cellAllChange"></el-checkbox> 
					<span><%=rb.getString("XiaoQuXinXi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.cell" class="col-group" v-model="form.cell">
					<el-checkbox v-for="item in cellCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.status,'el-icon-open':!expanded.status}" @click="expanded.status = !expanded.status"></i>
					<el-checkbox :indeterminate="form.status.length<statusCol.length" v-model="statusAll" @change="statusAllChange"></el-checkbox> 
					<span><%=rb.getString("ZhuangTai")%></span>
				</div>
				<el-checkbox-group v-show="expanded.status" class="col-group" v-model="form.status">
					<el-checkbox v-for="item in statusCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.network,'el-icon-open':!expanded.network}" @click="expanded.network = !expanded.network"></i>
					<el-checkbox :indeterminate="form.network.length<networkCol.length" v-model="networkAll" @change="networkAllChange"></el-checkbox> 
					<span><%=rb.getString("WangLuoSheZhi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.network" class="col-group" v-model="form.network">
					<el-checkbox v-for="item in networkCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.location,'el-icon-open':!expanded.location}" @click="expanded.location = !expanded.location"></i>
					<el-checkbox :indeterminate="form.location.length<locationCol.length && form.location.length>0" v-model="locationAll" @change="locationAllChange"></el-checkbox> 
					<span><%=rb.getString("WeiZhi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.location" class="col-group" v-model="form.location">
					<el-checkbox v-for="item in locationCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.satellite,'el-icon-open':!expanded.satellite}" @click="expanded.satellite = !expanded.satellite"></i>
					<el-checkbox v-model="satelliteAll" @change="satelliteAllChange"></el-checkbox> 
					<span><%=rb.getString("GPSWeiXingShu")%></span>
				</div>
				<el-checkbox-group v-show="expanded.satellite" class="col-group" v-model="form.satellite">
					<el-checkbox v-for="item in satelliteCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
		</div>
		
		<div style="border-left: 1px solid #e9e9e9;height:100%;flex:1;">
			<div class="select-all-cls" style="margin: 10px; padding: 0px; border-bottom: 1px solid #e9e9e9;">
				<el-checkbox v-if="false" :indeterminate="!dragAll" v-model="dragAll" @change="dragAllChange"></el-checkbox> 
				<span v-if="false"><%=rb.getString("QuanXuan")%></span>
				<span style="padding-bottom: 10px;margin-left: 0px;font-weight: bold;"><%=rb.getString("LiePaiXu")%></span>
			</div>
			<el-checkbox-group v-model="dragCol">
				<draggable
					class="list-group"
					v-model="columnsBase"
					v-bind="dragOptions">
					<transition-group type="transition" :name="!drag? 'flip-list':null">
						<div v-for="(col,idx) in columnsBase" :key="col.field" class="list-group-item" v-if="showCols.includes(col.field)">
							<span class="el-checkbox__label" style="border-bottom: 1px dashed #e9e9e9;min-width: 260px;color: #606266;">
								{{col.field === 'remark' ? (enbvm && enbvm.currentRemarkLabel ? enbvm.currentRemarkLabel : 'Remark') : col.label}}

								<i v-if="!col.disabled" class="el-icon el-icon-close sort-item-op" style="zoom: 0.6;float: right; margin-top: 6px;" @click="clickColumnLabel(col.field === 'remark' ? (enbvm && enbvm.currentRemarkLabel ? enbvm.currentRemarkLabel : 'Remark') : col.label)"></i>
							</span>
							<el-checkbox v-if="false" :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
						</div>
					</transition-group>
				</draggable>
			</el-checkbox-group>
		</div>
	</div>
	<div class="windowButtonGroup" style="float:none !important;padding:20px 0 20px 30px;position:relative;z-index:321;">			
		<a class="linkbutton linkbutton_trend" @click="ColumnConfigEn()"><span><%=rb.getString("QueDing")%></span></a>
		<a class="linkbutton linkbutton_nowanna" @click="close"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
<script>
var idGlobal = 3;
new Vue({
	el: '#showOrHideItem',
	data() {

		var ignores = [];

		if("${isSuperAdmin}" != '1'){
			ignores.push('IPSEC_ADDR');
			ignores.push('mmepool_ipsec_addr');
		}
		/* 以下字段受此参数('${enbAdditionalColShow}')控制显示或隐藏 */
		if(!(supportTopoSite || '${enbAdditionalColShow}' == 'true')) {
			ignores.push('sub_station_name');
		}
        if('${enbAdditionalColShow}' != 'true' && registerSiteIdShow != 'true'){
            ignores.push('site_id');
        }
		if('${enbAdditionalColShow}' != 'true'){
			
			ignores.push('install_address');
			ignores.push('circuit_ref');
			ignores.push('circuit_jo');
			ignores.push('service_status');
			ignores.push('rom');
			ignores.push('contact_number');
		}
		if (isCloud != 'true' && false){
			ignores.push('available_rate');
		}
		//cloud版的支持halob 
		if( isSupportHalob != 'true'){
			ignores.push('halob_flag');
		}
		if(writableMap["CODE_ENB_EXPIRY_DATE"] == undefined){
			ignores.push('validity');
			//ignores.push('lock_status');
		}

		return {
			ignores: ignores,
			dragCol: [
				// 'serial_number',
				// 'host_name',
				'rf_status',
				'op_state',
				'CELL_IDENTITY',
				'PHYCELLID',
				'mme_status',
				'ue_count',
				'cpe_connect',
				'cell_ip',
				'mac_address',
				'product',
				'module_type',
				'software_version'
			],
			drag: false,
			columnsBase: [
				// 用于存储columns的基础配置（不包含动态label）
				// {field: 'serial_number', label: '<%=rb.getString("XiaoZhanBianMa")%>',sortable: true,disabled: true, width: 180},
                // {field: 'host_name', label: '<%=rb.getString("HostName")%>',sortable: true,disabled: true, width: 150},
				{field: 'enbId', label: '<%=rb.getString("EnodebId")%>', width: 100},
				{field: 'cellId', label: '<%=rb.getString("XIAOQUID")%>', width: 100},
				{field: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>',sortable: true,disabled: true, width: 150},
				{field: 'op_state', label: '<%=rb.getString("ShiFouJiHuo")%>',sortable: true,disabled: true, width: 120},
				{field: 'CELL_IDENTITY', label: 'ECI',sortable: true,disabled: true, width: 100},
				{field: 'PHYCELLID', label: '<%=rb.getString("PCI2")%>',sortable: true,disabled: true, width: 70},
				{field: 'mme_status', label: '<%=rb.getString("MMEZhuangTai")%>',disabled: true, width: 120},
				{field: 'plmnid', label: '<%=rb.getString("PLMN")%>', width: 70},
				{field: 'bandwidth', label: '<%=rb.getString("DaiKuan")%>', width: 80},
				{field: 'ue_count', label: '<%=rb.getString("UEShu")%>',sortable: true,disabled: true, width: 80},
				
				{field: 'euCountStr', label: '<%=rb.getString("EUShu")%>',sortable: true, width: 80},
				{field: 'ruCountStr', label: '<%=rb.getString("RUShu")%>',sortable: true, width: 80},
				
				{field: 'cpe_connect', label: '<%=rb.getString("CPELianJieShu")%>',disabled: true, width: 100},
				{field: 'wanSpeed', label: '<%=rb.getString("WanZhuangTai")%>', width: 180},
				{field: 'cell_ip', label: 'IP',sortable: true,disabled: true, width: 120},
				{field: 'mac_address', label: 'MAC',sortable: true,disabled: true, width: 130},
				{field: 'product', label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',sortable: true,disabled: true, width: 110},
				{field: 'product_name', label: '<%=rb.getString("ChanPinMingCheng")%>',sortable: true, width: 120},
				{field: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>',sortable: true,disabled: true, width: 120},
				{field: 'software_version', label: '<%=rb.getString("SoftwareVersion")%>',sortable: true,disabled: true, width: 140},
				{field: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',sortable: true,disabled: true, width: 130},
				{field: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>', width: 170},
				{field: 'mmepool_ipsec_addr', label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>', width: 120},
				{field: 'site_id', label: siteIdLabel,sortable: true, width: 80},
				
				{field: 'sub_station_name', label: siteNameLabel, width: 120},
                {field: 'install_address', label: '<%=rb.getString("AnZhuangXiangXiDiZhi")%>', width: 200},
                {field: 'circuit_ref', label: 'Circuit Ref.', width: 120},
                {field: 'circuit_jo', label: 'Circuit J & O', width: 120},
                {field: 'service_status', label: '<%=rb.getString("ZhuangTai")%>',width: 120},
                {field: 'rom', label: 'Rom', width: 120},
                {field: 'contact_number', label: '<%=rb.getString("YeZhuLianXiFangShi")%>', width: 120},
                
				{field: 'EARFCNDLINUSE', label: '<%=rb.getString("PinDian")%>', width: 130},
				{field: 'synStatus', label: '<%=rb.getString("TongBuZhuangTai")%>',sortable: true, width: 160},
				{field: 'pm_report_status', label: '<%=rb.getString("KPIShangBaoZhuangTai")%>',sortable: true, width: 135},
				{field: 'gps_satellite_count', label: '<%=rb.getString("GPSWeiXingShu")%>',sortable: true, width: 85},
				{field: 'online_duration', label: '<%=rb.getString("LeiJiShiChang")%>',sortable: true, width: 120},
				{field: 'up_time', label: '<%=rb.getString("YunXingShiJian")%>',sortable: true, width: 120},
				{field: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>',sortable: true, width: 140},
				{field: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>',sortable: true, width: 140},
				{field: 'online_time', label: '<%=rb.getString("JieRuShiJian")%>',sortable: true, width: 140},
				{field: 'offline_time', label: '<%=rb.getString("DuanKaiShiJian")%>',sortable: true, width: 140},
				{field: 'network_model', label: '<%=rb.getString("JiZhanZhiShi")%>',sortable: true, width: 120},
				{field: 'firmware_version', label: '<%=rb.getString("FirmwareVersion")%>',sortable: true, width: 135},
				{field: 'gps_version', label: '<%=rb.getString("GPSBanBen")%>', width: 120},
				{field: 'available_rate', label: '<%=rb.getString("XiaoQuKeYongZhanBi")%>', width: 70},
				{field: 'halob_flag', label: 'HaloX',sortable: true, width: 100},
				{field: 'tac', label: '<%=rb.getString("TAC")%>', width: 80},
				{field: 'signment', label: '<%=rb.getString("ZiZhenPeiBi")%>', width: 80},
				{field: 'specialSubframe', label: '<%=rb.getString("TeShuZiZhenPeiBi")%>', width: 80},
				{field: 'rootIndex', label: '<%=rb.getString("GenXuLieSuoYin")%>', width: 80},
				{field: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>', width: 100},
				{field: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>', width: 90},
				{field: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>', width: 70},

				{field: 'mechanical_downtilt', label: '<%=rb.getString("JiXieXiaQingJiao")%>', width: 135},
				{field: 'electronic_downtilt', label: '<%=rb.getString("DianZiXiaQingJiao")%>', width: 135},
				{field: 'vertical_3dB_beam_width', label: '<%=rb.getString("ChuiZhiBoSuKuanDu")%>', width: 160},
				{field: 'horizontal_azimuth', label: '<%=rb.getString("ShuiPinFangWeiJiao")%>', width: 135},

				{field: 'validity', label: '<%=rb.getString("YouXiaoQi")%>', width: 125},
				{field: 'lock_status', label: '<%=rb.getString("SuoDingZhuangTai")%>', width: 90},
				{field: 'tx_power', label: '<%=rb.getString("CPETxPower")%>', width: 80},
                {field: 'remark', label: 'Remark', width: 140}, // label会通过computed动态更新
                // {field: 'authCode', label: '<%=rb.getString("JianQuanMa")%>', width: 120}
			],
			form: {
				device: ['serial_number','product','module_type','software_version','mac_address','group_name','online_time','offline_time'],
				cell: ['host_name','CELL_IDENTITY','PHYCELLID'],
				status: ['op_state','mme_status','rf_status','ue_count','cpe_connect'],
				network: ['cell_ip'],
				location: [],
				satellite: []
			},
			expanded: {
				device: true,
				cell: true,
				status: true,
				network: true,
				location: true,
				satellite: true
			}
		};
	},
	computed: {
		cellCol() {
			var vm = this;
			var cols = [
            	{code: 'enbId', label: '<%=rb.getString("EnodebId")%>', width: 100},
				{code: 'host_name', label: '<%=rb.getString("HostName")%>',disabled: true},
                {code: 'cellId', label: '<%=rb.getString("XIAOQUID")%>', width: 100},
				{code: 'CELL_IDENTITY', label: 'ECI',disabled: true},
				{code: 'PHYCELLID', label: '<%=rb.getString("PCI2")%>',disabled: true},
				{code: 'plmnid', label: '<%=rb.getString("PLMN")%>'},
				{code: 'tac', label: '<%=rb.getString("TAC")%>'},
				{code: 'signment', label: '<%=rb.getString("ZiZhenPeiBi")%>'},
				{code: 'specialSubframe', label: '<%=rb.getString("TeShuZiZhenPeiBi")%>'},
				{code: 'rootIndex', label: '<%=rb.getString("GenXuLieSuoYin")%>'},
				{code: 'site_id', label: siteIdLabel},
				{code: 'bandwidth', label: '<%=rb.getString("DaiKuan")%>'},
				{code: 'EARFCNDLINUSE', label: '<%=rb.getString("PinDian")%>'},
				{code: 'network_model', label: '<%=rb.getString("JiZhanZhiShi")%>'},
				{code: 'tx_power',label:'<%=rb.getString("CPETxPower")%>'},
				
				{code: 'circuit_ref', label: 'Circuit Ref.'},
                {code: 'circuit_jo', label: 'Circuit J & O'},
                {code: 'contact_number', label: '<%=rb.getString("YeZhuLianXiFangShi")%>'}
			];
			return cols.filter(function(item){ return !vm.ignores.includes(item.code);});
		},
		statusCol() {
			var vm = this;
			var cols = [
				{code: 'op_state', label: '<%=rb.getString("ShiFouJiHuo")%>',disabled: true},
				{code: 'mme_status', label: '<%=rb.getString("MMEZhuangTai")%>',disabled: true},
				{code: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>',disabled: true},
				{code: 'pm_report_status', label: '<%=rb.getString("KPIShangBaoZhuangTai")%>'},
				{code: 'halob_flag', label: 'HaloX'},
				{code: 'synStatus', label: '<%=rb.getString("TongBuZhuangTai")%>'},
				{code: 'validity', label: '<%=rb.getString("YouXiaoQi")%>'},
				{code: 'lock_status', label: '<%=rb.getString("SuoDingZhuangTai")%>'},
				{code: 'ue_count', label: '<%=rb.getString("UEShu")%>',disabled: true},
				{code: 'euCountStr', label: '<%=rb.getString("EUShu")%>'},
				{code: 'ruCountStr', label: '<%=rb.getString("RUShu")%>'},
				{code: 'cpe_connect', label: '<%=rb.getString("CPELianJieShu")%>',disabled: true},
				{code: 'wanSpeed', label: '<%=rb.getString("WanZhuangTai")%>'},
				
				{code: 'service_status', label: '<%=rb.getString("ZhuangTai")%>'}
			];
			return cols.filter(function(item){ return !vm.ignores.includes(item.code);});
		},
		networkCol() {
			var vm = this;
			var cols = [
				{code: 'mmepool_ipsec_addr', label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>'},
				{code: 'cell_ip', label: '<%=rb.getString("IPDiZhi")%>',disabled: true},
				{code: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>'}
			];
			return cols.filter(function(item){ return !vm.ignores.includes(item.code);});
		},
		deviceCol() {
			var vm = this;
			var cols = [
				{code: 'serial_number', label: '<%=rb.getString("XiaoZhanBianMa")%>',disabled: true},
				{code: 'product', label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',disabled: true},
				{code: 'product_name', label: '<%=rb.getString("ChanPinMingCheng")%>'},
				{code: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>',disabled: true},
				{code: 'software_version', label: '<%=rb.getString("SoftwareVersion")%>',disabled: true},
				{code: 'firmware_version', label: '<%=rb.getString("FirmwareVersion")%>'},
				{code: 'online_duration', label: '<%=rb.getString("LeiJiShiChang")%>'},
				{code: 'up_time', label: '<%=rb.getString("YunXingShiJian")%>'},
				{code: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},				
				{code: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
				{code: 'online_time', label: '<%=rb.getString("JieRuShiJian")%>'},
				{code: 'offline_time', label: '<%=rb.getString("DuanKaiShiJian")%>'},
				{code: 'mac_address', label: '<%=rb.getString("XiaoZhanMAC")%>',disabled: true},
				{code: 'gps_version', label: '<%=rb.getString("GPSBanBen")%>'},
				{code: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',disabled: true},
                // {code: '<%=rb.getString("JianQuanMa")%>',label:'authCode'},
				{code: 'sub_station_name', label: siteNameLabel},
				{code: 'rom', label: 'Rom'},
                {code: 'remark', label: enbvm.currentRemarkLabel},
			];
			return cols.filter(function(item){ return !vm.ignores.includes(item.code);});
		},
		locationCol() {
			var vm = this;
			var cols = [
				{code: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>'},
				{code: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>'},
				{code: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>'},

				{code: 'mechanical_downtilt', label: '<%=rb.getString("JiXieXiaQingJiao")%>'},
				{code: 'electronic_downtilt', label: '<%=rb.getString("DianZiXiaQingJiao")%>'},
				{code: 'vertical_3dB_beam_width', label: '<%=rb.getString("ChuiZhiBoSuKuanDu")%>'},
				{code: 'horizontal_azimuth', label: '<%=rb.getString("ShuiPinFangWeiJiao")%>'},
				
				{code: 'install_address', label: '<%=rb.getString("AnZhuangXiangXiDiZhi")%>'}
			];
			return cols.filter(function(item){ return !vm.ignores.includes(item.code);});
		},
		satelliteCol() {
			return [
				{code: 'gps_satellite_count', label: '<%=rb.getString("GPSWeiXingShu")%>'}
			];
		},
		columns() {
			var vm = this;
			// 基于columnsBase创建动态columns数组
			return vm.columnsBase.map(function(col) {
				// 如果是remark列，使用动态的label
				if(col.field === 'remark') {
					return Object.assign({}, col, {
						label: enbvm && enbvm.currentRemarkLabel ? enbvm.currentRemarkLabel : 'Remark'
					});
				}
				// 其他列保持不变
				return col;
			});
		},
		dragOptions() {

			return {
				animation: 200,
				group: 'description',
				disabled: false,
				ghostClass: 'ghost'
			};
		},
		dragAll() {
			var vm = this;
			return vm.dragCol.length == vm.columns.length;
		},
		colAll() {
			var vm = this;
			return vm.deviceAll && vm.cellAll && vm.statusAll && vm.locationAll && vm.satelliteAll;
		},
		deviceAll() {
			var vm = this;
			return vm.deviceCol.length == vm.form.device.length;
		},
		cellAll() {
			var vm = this;
			return vm.cellCol.length == vm.form.cell.length;
		},
		statusAll() {
			var vm = this;
			return vm.statusCol.length == vm.form.status.length;
		},
		networkAll() {
			var vm = this;
			return vm.networkCol.length == vm.form.network.length;
		},
		locationAll() {
			var vm = this;
			return vm.locationCol.length == vm.form.location.length;
		},
		satelliteAll() {
			var vm = this;
			return vm.satelliteCol.length == vm.form.satellite.length;
		},
		showCols() {
			var vm = this;

			return vm.form.device.concat(vm.form.cell).concat(vm.form.status).concat(vm.form.network).concat(vm.form.location).concat(vm.form.satellite);
			//return vm.dragCol;
		}
	},
	methods: {
		clickColumnLabel(label) {
			$('.el-checkbox__label:contains('+label+')').click();
		},
		dragAllChange(val) {
			var vm = this,
				fields = vm.columns.map(function(item){
					return item.field;
				}),
				filters = vm.columns.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.field;
				});
			
			vm.dragCol = val?fields:filters;
		},
		colAllChange(val) {
			var vm = this;

			vm.deviceAllChange(val);
			vm.cellAllChange(val);
			vm.statusAllChange(val);
			vm.networkAllChange(val);
			vm.locationAllChange(val);
			vm.satelliteAllChange(val);
		},
		deviceAllChange(val) {
			var vm = this,
				fields = vm.deviceCol.map(function(item){
					return item.code;
				}),
				filters = vm.deviceCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.device = val?fields:filters;
		},
		cellAllChange(val) {
			var vm = this,
				fields = vm.cellCol.map(function(item){
					return item.code;
				}),
				filters = vm.cellCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.cell = val?fields:filters;
		},
		statusAllChange(val) {
			var vm = this,
				fields = vm.statusCol.map(function(item){
					return item.code;
				}),
				filters = vm.statusCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.status = val?fields:filters;
		},
		networkAllChange(val) {
			var vm = this,
				fields = vm.networkCol.map(function(item){
					return item.code;
				}),
				filters = vm.networkCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.network = val?fields:filters;
		},
		locationAllChange(val) {
			var vm = this,
				fields = vm.locationCol.map(function(item){
					return item.code;
				}),
				filters = vm.locationCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.location = val?fields:filters;
		},
		satelliteAllChange(val) {
			var vm = this,
				fields = vm.satelliteCol.map(function(item){
					return item.code;
				}),
				filters = vm.satelliteCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.satellite = val?fields:filters;
		},
		close() {
			$(".showHideItem").slideUp(500);
		},
		ColumnConfigEn() {
			var vm = this,
				url = '${ctx}/cell/cpeinfos/cellColumnConfig.action',
				sortCol = [];
				
			vm.columns.map(function(item){
				sortCol.push(item.field);
			});

			vm.configColumn();

			url = '${ctx}/system/column/setting/insert.action';
			// 保存显示列
			var params = {
					pageName: '1',
					showColumn: vm.showCols.join(','),
					sortColumn: sortCol.join(',')
				};

			axios.post(url, params).then(function(res){
				enbvm.columns.splice(0, enbvm.columns.length);
				vm.columns.map(function(col){
					enbvm.columns.push(Object.assign({},col));
				});

				var tbStates = enbvm.$refs.list.$refs.ctableInner.store.states,
					sortCodes = sortCol,
					storeCols = tbStates.columns;

				tbStates._columns = storeCols.sort(function(n, m) {
					var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
						idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);

					if( [undefined,'enb_operation','connection_status','alarm','serial_number','host_name'].includes(n.property) ) {
						idxn = 0;
					}
					if( [undefined,'enb_operation','connection_status','alarm','serial_number','host_name'].includes(m.property) ) {
						idxm = 0;
					}

					return idxn - idxm;
				});

				enbvm.$refs.list.$refs.ctableInner.store.updateColumns();
				$('#cellInfo').css('width','99.9%');
				setTimeout(function(){
					$('#cellInfo').css('width','100%');
				},2000);
			}).catch(function(){});

			vm.close();
		},
		configColumn() {
			var vm = this,
				eNodeB_column = enbvm.getAllDefaultCols();

			var columns = eNodeB_column.filter(function(item){
				return vm.showCols.includes(item.prop) || ['oepr','enb_operation','connection_status','alarm','serial_number','host_name'].includes(item.prop);
			});

			if(enableCheckbox){
				columns.unshift({field:'oepr',checkbox:true,width:50});
			}

			enbvm.showProps = vm.showCols;
		}
	},
	created() {
		var vm = this,
			map = {
				device: vm.deviceCol.map(function(item){ return item.code;}),
				cell: vm.cellCol.map(function(item){ return item.code;}),
				status: vm.statusCol.map(function(item){ return item.code;}),
				network: vm.networkCol.map(function(item){ return item.code;}),
				location: vm.locationCol.map(function(item){ return item.code;}),
				satellite: vm.satelliteCol.map(function(item){ return item.code;})

			},
			dragCodes = vm.columns.map(function(item){ return item.field}),
			showCols = enbvm.showProps;

		showCols.map(function(col){
			['device','cell','status','network','location','satellite'].map(function(code){
				if(map[code].includes(col) && !vm.form[code].includes(col)) vm.form[code].push(col);
			});
			// 初始化默认选中项
			if(dragCodes.includes(col) && !vm.dragCol.includes(col)) vm.dragCol.push(col);
		});
		// 假设为排序 --->
		var sortCodes = enbvm.sortColumns,
			sortList = vm.columnsBase;

		vm.columnsBase = sortList.sort(function(n, m){
			var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
				idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

			return idxn - idxm;
		});
		// <--- 排序

		vm.configColumn();
	},
	mounted() {
		//setTimeout(closeLoading, 500);
		this.ColumnConfigEn();
	}
});
</script>