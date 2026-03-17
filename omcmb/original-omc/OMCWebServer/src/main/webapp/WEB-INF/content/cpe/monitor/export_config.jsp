<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
	.columns-list-cls {
		max-width: 800px;
	}
	.columns-list-cls .el-checkbox {
		min-width: 210px;
	}
	#cpe_export_form .el-checkbox__label {
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
	.tmp-chart {
		width: 800px;
		height: 300px;
		position: absolute;
		top: 40px;
		left: 300px;
		background: #fff !important;
		z-index: 1000;
		box-shadow: 1px 2px 5px silver;
	}
	.gray-color::before {
		font-size: 12px;
		color: #999;
	}
	#cpe_export_form .el-icon-close1::before {
		color: #333;
	}
</style>

<el-form ref="form" id="cpe_export_form">
	<div class="flex-ctn" style="padding-left: 20px;">
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
				<i :class="{'el-icon':true,'el-icon-close1':expanded.lte,'el-icon-open':!expanded.lte}" @click="expanded.lte = !expanded.lte"></i>
				<el-checkbox :indeterminate="form.lte.length<lteCol.length" v-model="lteAll" @change="lteAllChange"></el-checkbox> 
				<span><%=rb.getString("LTEZhuangTai")%></span>
			</div>
			<el-checkbox-group v-show="expanded.lte" class="col-group" v-model="form.lte">
				<el-checkbox v-for="item in lteCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
			</el-checkbox-group>
		</div>
		<div>
			<div class="select-all-cls">
				<i :class="{'el-icon':true,'el-icon-close1':expanded.nr,'el-icon-open':!expanded.nr}" @click="expanded.nr = !expanded.nr"></i>
				<el-checkbox :indeterminate="form.nr.length<nrCol.length && form.nr.length>0" v-model="nrAll" @change="nrAllChange"></el-checkbox> 
				<span>NR Status</span>
			</div>
			<el-checkbox-group v-show="expanded.nr" class="col-group" v-model="form.nr">
				<el-checkbox v-for="item in nrCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
			</el-checkbox-group>
		</div>
		<div>
			<div class="select-all-cls">
				<i :class="{'el-icon':true,'el-icon-close1':expanded.lan,'el-icon-open':!expanded.lan}" @click="expanded.lan = !expanded.lan"></i>
				<el-checkbox :indeterminate="form.lan.length<lanCol.length" v-model="lanAll" @change="lanAllChange"></el-checkbox> 
				<span><%=rb.getString("LANZhuangTai")%></span>
			</div>
			<el-checkbox-group v-show="expanded.lan" class="col-group" v-model="form.lan">
				<el-checkbox v-for="item in lanCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
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
	</div>
	<div style="padding: 20px 20px;">
		<el-button type="primary" @click="comfirmExport"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="close"><%=rb.getString("QuXiao")%></el-button>
		<span style="color: 999;font-weight: normal;margin-left: 20px;"><i class="el-icon el-icon-circle-info gray-color"></i> <%=rb.getString("DengDaiTiShi")%></span>
	</div>
</el-form>

<script>
	new Vue({
		el: '#cpe_export_form',
		data() {

			return {
				disabledCol: ['MACADDRESS','CPE_NAME','IMSI','IPADDRESS','SOFTWARE_VERSION','group_name','cpe_model'],
				colList: ['MACADDRESS','CPE_NAME','IMSI','IPADDRESS','SOFTWARE_VERSION','group_name','cpe_model'],
				allOp: false,
				base64: {

				},
				deviceCol: [
					{code: 'SERIAL_NUMBER', label: '<%=rb.getString("CPEBianMa")%>',disabled: true},
					{code: 'CPE_NAME', label: '<%=rb.getString("CPEName")%>',disabled: true},
					{code: 'MODEL_NAME', label: '<%=rb.getString("ChanPinXingHao")%>',disabled: true},
					{code: 'PRODUCT', label: '<%=rb.getString("ChanPinLeiXing")%>'},
					{code: 'SOFTWARE_VERSION', label: '<%=rb.getString("CPEVersion")%>',disabled: true},
					{code: 'UPTIME', label: '<%=rb.getString("YunXingShiJian")%>'},
					{code: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
					{code: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
					{code: 'MCC', label: 'MCC'},
					{code: 'MNC', label: 'MNC'},
					{code: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',disabled: true},
					{code: 'module_name', label: '<%=rb.getString("MoKuaiMingCheng")%>'},
					{code: 'module_version', label: '<%=rb.getString("MoKuaiBanNen")%>'},
					{code: 'LGW_IP', label: '<%=rb.getString("LgwIPAddress")%>'},
                    {code: 'MARKET_NAME', label: '<%=rb.getString("ChanPinMingCheng")%>', width: 150},
					{code: 'IMEI', label: 'IMEI', width: 100},
					{code: 'cpe_model', label: '<%=rb.getString("SheBeiXingHao")%>', width: 120,disabled: true}
				],
				lteCol: [
					{code: 'IMSI', label: 'IMSI',disabled: true},
					{code:'SCANMODE',label:'<%=rb.getString("SaoMiaoFangShi")%>'},
					{code: 'PCI', label: 'PCI',disabled: true},
					{code: 'HOST_NAME', label: '<%=rb.getString("HostName")%>',disabled: true},
					{code: 'CELL_IDENTITY', label: 'ECI',disabled: true},
					{code: 'DL_EARFCN', label: '<%=rb.getString("PinDian")%>'},
					{code: 'BANDWIDTH', label: '<%=rb.getString("DaiKuan")%>(MHz)'},
					{code: 'CINR0', label: 'CINR1'},
					{code: 'CINR1', label: 'CINR2'},
					{code: 'CPE_SINR', label: 'SINR'},
					{code: 'DL_CURRENT_DATARATE', label: '<%=rb.getString("CPEXiaXingTunTuLiang")%>'},
					{code: 'UL_CURRENT_DATARATE', label: '<%=rb.getString("CPEShangXingTunTuLiang")%>'},
					{code: 'TX_POWER', label: '<%=rb.getString("CPETxPower")%>'},
					{code: 'RSRP0', label: 'RSRP1'},
					{code: 'RSRP1', label: 'RSRP2'},
					{code: 'UL_MCS', label: 'UL_MCS'},
					{code: 'DL_MCS', label: 'DL_MCS'},
					{code:'LTE_CONNECTION_TIME',label:'<%=rb.getString("LTEGengXinShiJian")%>'},
					{code:'DL_BLER',label:'DL BLER'},
				],
				nrCol: [
					{code: 'NR_BAND', label: 'NR-Band'},
					{code: 'NR_BANDWIDTH', label: 'NR-<%=rb.getString("DaiKuan")%>(MHz)'},
					{code: 'NR_PCI', label: 'NR-PCI'},
					{code: 'NR_EARFCN', label: 'NR-<%=rb.getString("PinDian")%>'},
					{code: 'NR_PLMN', label: 'NR-PLMN'},
					{code: 'NR_CELL_ID', label: 'NR-Cell ID'},
					{code: 'NR_DL_FREQUENCY', label: 'NR-DL_Frequency'},
					{code: 'NR_UL_FREQUENCY', label: 'NR-UL_Frequency'},
					{code: 'NR_CINR', label: 'NR-CINR'},
					{code: 'NR_SINR', label: 'NR-SINR'},
					{code: 'NR_RSRQ', label: 'NR-RSRQ'},
					{code: 'NR_RSRP', label: 'NR-RSRP'},
				],
				lanCol: [
					{code: 'MACADDRESS', label: '<%=rb.getString("CPEMacAddress")%>',disabled: true},
					{code: 'IPADDRESS', label: '<%=rb.getString("CPEIPAddress")%>',disabled: true},
					{code: 'LGW_MAC', label: '<%=rb.getString("LgwMacAddress")%>'}
				],
				locationCol: [
					{code: 'longitude', label: '<%=rb.getString("JingDu")%>'},
					{code: 'latitude', label: '<%=rb.getString("WeiDu")%>'},
					{code: 'height', label: '<%=rb.getString("GaoDu")%>'},
					{code: 'distance', label: '<%=rb.getString("JuLi")%>'},
				],
				form: {
					device: ['SERIAL_NUMBER','CPE_NAME','MODEL_NAME','SOFTWARE_VERSION','group_name','cpe_model'],
					lte: ['IMSI','PCI','HOST_NAME','CELL_IDENTITY'],
					nr:[],
					lan: ['MACADDRESS','IPADDRESS'],
					location: []
				},
				expanded: {
					device: true,
					lte: true,
					nr: true,
					lan: true,
					location: true
				}
			};
		},
		computed: {
			chartOptions() {
				var vm = this;

				return vm.allCharts.map(function(item){
					return item.code;
				})
			},
			tbIndeter() {
				var vm = this,
					cpe_column = cpevm.getAllColumns();

				return vm.colList.length && vm.colList.length < cpe_column.length;
			},
			allTb() {
				var vm = this,
					cpe_column = cpevm.getAllColumns();

				return vm.colList.length == cpe_column.length;
			},
			colAll() {
				var vm = this;
				return vm.deviceAll && vm.lteAll && vm.nrAll && vm.lanAll && vm.locationAll;
			},
			deviceAll() {
				var vm = this;
				return vm.deviceCol.length == vm.form.device.length;
			},
			lteAll() {
				var vm = this;
				return vm.lteCol.length == vm.form.lte.length;
			},
			nrAll() {
				var vm = this;
				return vm.nrCol.length == vm.form.nr.length;
			},
			lanAll() {
				var vm = this;
				return vm.lanCol.length == vm.form.lan.length;
			},
			locationAll() {
				var vm = this;
				return vm.locationCol.length == vm.form.location.length;
			},
			showCols() {
				var vm = this;
				return vm.form.device.concat(vm.form.lte).concat(vm.form.nr).concat(vm.form.lan).concat(vm.form.location);
			}
		},
		methods: {
			comfirmExport() {
				var vm = this,
					cpe_column = cpevm.getAllColumns(),
					params = {
						TimeZone: timeZone,
						operator_codes: '${operator_code}',
						content: ''
					};
				
				var columns = cpe_column.filter(function(item){
						if(item.prop == 'lte_connection_time'){
							item.prop = 'LTE_CONNECTION_TIME'
						}
						return vm.showCols.includes(item.prop) || ['CONNECTION_STATUS'].includes(item.prop);
					}),
					content = columns.map(function(item){ return item.prop;}).join(',');
	
				params.content = content;
				
				var queryParams = cpevm.queryParams;
					
				Object.assign(params, queryParams);
				exportByForm('${ctx}/cell/CPE/exportCpesToExcel.action', params);
				vm.close();
			},
			close() {
				document.body.click();
			},
			colAllChange(val) {
				var vm = this;

				vm.deviceAllChange(val);
				vm.lteAllChange(val);
				vm.nrAllChange(val);
				vm.lanAllChange(val);
				vm.locationAllChange(val);
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
			lteAllChange(val) {
				var vm = this,
					fields = vm.lteCol.map(function(item){
						return item.code;
					}),
					filters = vm.lteCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.lte = val?fields:filters;
			},
			nrAllChange(val) {
				var vm = this,
					fields = vm.nrCol.map(function(item){
						return item.code;
					}),
					filters = vm.nrCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.nr = val?fields:filters;
			},
			lanAllChange(val) {
				var vm = this,
					fields = vm.lanCol.map(function(item){
						return item.code;
					}),
					filters = vm.lanCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.lan = val?fields:filters;
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
			}
		},
		mounted() {
			
		}
	});

</script>
