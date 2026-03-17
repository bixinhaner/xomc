<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
	.columns-list-cls {
		max-width: 920px;
	}
	.columns-list-cls .el-checkbox {
		min-width: 210px;
	}
	#enb_export_form .el-checkbox__label {
		font-size: 12px;
		font-weight: normal;
	}
	.select-all-cls {
		padding: 10px 0 0 9px;
		display: flex;
		align-items: center;
	}
	.select-all-cls > span {
		font-weight: bold;
		margin-left: 10px;
	}
	.select-all-cls > i {
		margin-right: 5px;
	}
	.gray-color::before {
		font-size: 12px;
		color: #999;
	}
	#enb_export_form .el-icon-close1::before {
		color: #333;
	}
</style>
<div id="enb_export_form">
    <div style="width: 100%;height:450px;overflow: auto;">
        <el-form ref="form" style="padding-top: 10px;" :model="opform" :rules="rules">
            <div v-show="operatorShow">
                <div class="select-all-cls">
                    <span style="margin-left: 10px;"><%=rb.getString("YiXuanZeYunYingShang")%></span>
                </div>
                <el-ctable ref="optb" :url="opUrl" row-key="operator_code" style="margin: 0px 20px;border: 1px solid #f3f3f3;" height="200" :default-checked="operList" @selection-change="handleChange">
                    <el-table-column prop="operator_code" type="selection" :reserve-selection="true"></el-table-column>
                    <el-table-column prop="operator_name" label="<%=rb.getString("YunYingShangMingCheng")%>"></el-table-column>
                </el-ctable>
                <el-form-item prop="operList" style="margin-left: 20px;">
                    <el-input type="hidden" v-model="opform.operList"></el-input>
                </el-form-item>
                <hr style="border: none;border-bottom: 1px solid #e4e7ec;margin: 10px 0;">
            </div>
            <div class="flex-ctn" style="padding-left: 10px;">
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

            <hr style="border: none;border-bottom: 1px solid #e4e7ec;margin: 10px 0;">
            <div style="margin-left: 20px;">
                <div style="font-size: 14px;color:#606266;font-weight:bold;"><%=rb.getString("LicenseXinXi")%></div>
                <el-checkbox v-model="isExportDeviceLicenseInfo" style="margin: 10px 20px;"><%=rb.getString("DaoChuLicenseXinXi")%></el-checkbox> 
            </div>

            <hr style="border: none;border-bottom: 1px solid #e4e7ec;margin: 10px 0;">
            <div style="margin-left: 20px;">
                <div style="font-size: 14px;color:#606266;font-weight:bold;"><%=rb.getString("DaoChuGeShi")%></div>
				<el-radio-group v-model="exportFormatType">
					<el-radio style="margin: 10px 20px;" label="csv"><%=rb.getString("CSVGeShi")%></el-radio> 
					<el-radio style="margin: 10px 20px;" label="xlsx"><%=rb.getString("XLSXGeShi")%></el-radio> 
				</el-radio-group>
            </div>
        </el-form>
    </div>
    <div style="padding: 20px;">
        <el-button type="primary" @click="comfirmExport"><%=rb.getString("QueDing")%></el-button>
        <el-button @click="close"><%=rb.getString("QuXiao")%></el-button>
        <span style="color: 999;font-weight: normal;margin-left: 20px;"><i class="el-icon el-icon-circle-info gray-color"></i> <%=rb.getString("DengDaiTiShi")%></span>
    </div>
</div>
<script>
	new Vue({
		el: '#enb_export_form',
		data() {
			var ignores = [],
				vm = this;

			if("${isSuperAdmin}" != '1'){
				ignores.push('IPSEC_ADDR');
				ignores.push('mmepool_ipsec_addr');
			}
			/* 目前只有Amara支持ups，后面后端会调整逻辑 */
			/*if(siteIdShow != 'true'){
				ignores.push('site_id');
			}*/
			/* 以下字段受此参数('${enbAdditionalColShow}')控制显示或隐藏 */
			if(!(supportTopoSite || '${enbAdditionalColShow}' == 'true')){
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
			if (isCloud != 'true'){
				ignores.push('available_rate');
			}
			//cloud版的支持halob 
			if( isSupportHalob != 'true'){
				ignores.push('halob_flag');
			}
			if(writableMap["CODE_ENB_EXPIRY_DATE"] == undefined){
				ignores.push('validity');
				ignores.push('lock_status');
			}

			var validOper = function(rule, value, cb) {
				if(vm.operatorShow) {
					if(value && value.length) {
						cb();
					}else {
						cb('<%=rb.getString("XuanZeYunYingShang")%>');
					}
				}else {
					cb();
				}
			};

			return {
				ignores: ignores,
				exportFormatType: 'xlsx',

				opform: {
					operList: []
				},
				rules: {
					operList: [
						//{required: true,message: '<%=rb.getString("XuanZeYunYingShang")%>'}
						{validator: validOper}
					]
				},
				operList: ['${operator_code}'],
				allOp: false,
				opUrl: '${ctx}/system/operator/getOperatorListByPage.action',
				base64: {

				},
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
				},
                isExportDeviceLicenseInfo: false
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
					{code: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>'},
					{code: 'cell_ip', label: '<%=rb.getString("IPDiZhi")%>',disabled: true}
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
					{code: 'sub_station_name', label: siteNameLabel},
					{code: 'rom', label: 'Rom'},
                    {code: 'remark', label: enbvm && enbvm.currentRemarkLabel ? enbvm.currentRemarkLabel : 'Remark'},
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
			isCloud() {
				return isCloud == 'true';
			},
			isSuperAdmin() {
				return is_super_user == 'true';
			},
			operatorShow() {
				return isCloud == 'true' && is_super_user == 'true';
			},
			tbOptions() {
				var vm = this;

				return eNodeB_column.map(function(item){
					return item.field;
				})
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
			}
		},
		methods: {
			handleChange(s) {
				this.opform.operList = s;
			},
			comfirmExport() {
				var vm = this,
					params = {
						TimeZone: timeZone,
						operator_codes: '',
						content: '',
						sort: enbvm.queryParams.sort,
						order: enbvm.queryParams.order
					};
				
				var allCols = enbvm.getAllDefaultCols(),
					columns = allCols.filter(function(item){
					
						return vm.showCols.includes(item.prop) || ['connection_status','alarm'].includes(item.prop);
					}),
					content = columns.map(function(item){ return item.prop;}).join(',');

				params.content = 'connection_status,' + content;

				if(vm.operatorShow) {// 云环境且为amdin才可以选择运营商
					if(vm.opform.operList) {
						params['operator_codes'] = vm.opform.operList.map(function(row){
							return row.operator_code;
						}).join(',');
					}
				}else {// 运营商和普通用户
					params['operator_codes'] = '${operator_code}';
				}

				var queryParams = enbvm.queryParams;

				var url = '${ctx}/cell/cpeinfos/exportCellsToExcel.action';
				if(vm.exportFormatType == 'csv') {
					url = '${ctx}/cell/cpeinfos/exportCellsToCsv.action';
				}

				Object.assign(params,queryParams);
				vm.$refs.form.validate(function(valid){
					if(valid) {
						downLoadFileByAxios(url, params);
                        setTimeout(function(){
                            if(vm.isExportDeviceLicenseInfo){
                                downLoadFileByAxios('${ctx}/cell/cpeinfos/exportEnodebLicenseInfos.action', params);
                            }
                        }, 1000);
						if(vm.exportFormatType != 'csv') {
                        	queryVue.queryProgress();
						}
						vm.close();
					}
				});
			},
			close() {
				document.body.click();
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
		},
		mounted() {
			
		}
	});

</script>
