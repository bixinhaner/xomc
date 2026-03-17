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
	#sync_params .el-checkbox__label {
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
	.gray-color {
		font-size: 12px;
		color: #999;
	}
	#sync_params .el-icon-close1::before {
		color: #333;
	}
</style>
<div id="sync_params">
	<div class="flex-ctn" style="padding-left: 10px;">
		<div class="select-all-cls" style="padding-left: 10px;">
			<el-checkbox v-model="alarmSync" ></el-checkbox> 
			<span><%=rb.getString("GaoJingGuanLi")%></span>
			<span class="gray-color">( <%=rb.getString("HuoDongGaoJing")%> )</span>
			
		</div>
		<div class="select-all-cls" style="padding-left: 10px;">
			<el-checkbox :indeterminate="colSel" v-model="colAll" @change="colAllChange"></el-checkbox> 
			<span style="margin-right:20px;"><%=rb.getString("JianCeCanShuMing")%></span>
			<div class="link-font" @click="expanded.params = !expanded.params">
				<span v-if="!expanded.params"><%=rb.getString("ZhanKai")%></span>
				<span v-else><%=rb.getString("GuanBi")%></span> 
			</div>
		</div>
		<div v-show="expanded.params">
			<div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.basic,'el-icon-open':!expanded.basic}" @click="expanded.basic = !expanded.basic"></i>
					<el-checkbox :indeterminate="form.basic.length<basicCol.length && form.basic.length>0" v-model="basicAll" @change="basicAllChange"></el-checkbox> 
					<span><%=rb.getString("JiChuPeiZhi")%></span>
				</div>
				<el-checkbox-group v-show="expanded.basic" class="col-group" v-model="form.basic">
					<el-checkbox v-for="item in basicCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
		<div>
			<div class="select-all-cls">
				<i :class="{'el-icon':true,'el-icon-close1':expanded.others,'el-icon-open':!expanded.others}" @click="expanded.others = !expanded.others"></i>
				<el-checkbox :indeterminate="form.others.length<othersCol.length && form.others.length>0" v-model="othersAll" @change="othersAllChange"></el-checkbox> 
				<span><%=rb.getString("GaoJiPeiZhi")%></span>
			</div>
			<el-checkbox-group v-show="expanded.others" class="col-group" v-model="form.others">
				<el-checkbox v-for="item in othersCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
			</el-checkbox-group>
		</div>
	</div>
	</div>
	<div style="padding: 20px;">
		<el-button type="primary" @click="comfirmSync"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="close"><%=rb.getString("QuXiao")%></el-button>
	</div>
</div>
<script>
	var syncParamVue = new Vue({
		el: '#sync_params',
		data() {
			var ignores = [],
				vm = this;

			if(is_super_user != 'true'){
				ignores.push('ipsec_addr');
				ignores.push('mme_addr');
			}
			
			//cloud版的支持halob 
			if( isSupportHalob != 'true'){
				ignores.push('halob_flag');
			}
			if(writableMap["CODE_ENB_EXPIRY_DATE"] == undefined){
			
				ignores.push('lease');
			}

			var 
				basicCol = [
					//{code: 'serial_number', label: '<%=rb.getString("XiaoZhanBianMa")%>',disabled: true},
					//{code: 'product', label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
					//{code: 'product_name', label: '<%=rb.getString("ChanPinMingCheng")%>'},
					{code: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>'},
					{code: 'software_version', label: '<%=rb.getString("SoftwareVersion")%>'},
					{code: 'firmware_version', label: '<%=rb.getString("FirmwareVersion")%>'},
					//{code: 'up_time', label: '<%=rb.getString("YunXingShiJian")%>'},
					//{code: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
					//{code: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
					{code: 'MAC', label: '<%=rb.getString("XiaoZhanMAC")%>'},
					//{code: 'gps_version', label: '<%=rb.getString("GPSBanBen")%>'},
					//{code: 'group_name', label: '<%=rb.getString("SheBeiZu")%>'},
					//{code: 'sub_station_name', label: '<%=rb.getString("ZhanZhiMingCheng")%>'},
					//{code: 'rom', label: 'Rom'},

					//{code: 'enbId', label: '<%=rb.getString("EnodebId")%>', width: 100},
					{code: 'cell_name', label: '<%=rb.getString("HostName")%>'},
					//{code: 'cellId', label: '<%=rb.getString("XIAOQUID")%>', width: 100},
					{code: 'ECI', label: 'ECI'},
					{code: 'PCI', label: '<%=rb.getString("PCI2")%>'},
					{code: 'plmn', label: '<%=rb.getString("PLMN")%>'},
					{code: 'tac', label: '<%=rb.getString("TAC")%>'},
					//{code: 'site_id', label: 'Shop ID'},
					{code: 'bandwidth', label: '<%=rb.getString("DaiKuan")%>'},
					{code: 'earfcn', label: '<%=rb.getString("PinDian")%>'},
					{code: 'duplex_mode', label: '<%=rb.getString("JiZhanZhiShi")%>'},
					{code: 'tx_power',label:'<%=rb.getString("CPETxPower")%>'},
					//{code: 'signment', label: '<%=rb.getString("ZiZhenPeiBi")%>'},
					
					//{code: 'circuit_ref', label: 'Circuit Ref.'},
					//{code: 'circuit_jo', label: 'Circuit J & O'},
					//{code: 'contact_number', label: '<%=rb.getString("YeZhuLianXiFangShi")%>'},
					{code: 'cell_status', label: '<%=rb.getString("ShiFouJiHuo")%>'},
					{code: 'mme_status', label: '<%=rb.getString("MMEZhuangTai")%>'},
					{code: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>'},
					//{code: 'pm_report_status', label: '<%=rb.getString("KPIShangBaoZhuangTai")%>'},
					{code: 'halob_flag', label: 'HaloX'},
					{code: 'sync_status', label: '<%=rb.getString("TongBuZhuangTai")%>'},
					{code: 'IP', label: '<%=rb.getString("IPDiZhi")%>'},
					{code: 'mme_addr', label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>'},
					//{code: 'validity', label: '<%=rb.getString("YouXiaoQi")%>'},
					{code: 'lease', label: '<%=rb.getString("SuoDingZhuangTai")%>'},
					{code: 'ue_count', label: '<%=rb.getString("UEShu")%>'},
					{code: 'root_sequence_index', label: '<%=rb.getString("GenXuLieSuoYin")%>'},
					{code: 'gps_satellites', label: '<%=rb.getString("GPSWeiXingShu")%>'},
					{code: 'sub_frame_assignment', label: '<%=rb.getString("ZiZhenPeiBi")%> + <%=rb.getString("TeShuZiZhenPeiBi")%>'},
					
					//{code: 'euCountStr', label: '<%=rb.getString("EUShu")%>'},
					//{code: 'ruCountStr', label: '<%=rb.getString("RUShu")%>'},
					//{code: 'cpe_connect', label: '<%=rb.getString("CPELianJieShu")%>'},
					{code: 'wan_speed', label: '<%=rb.getString("WanZhuangTai")%>'},
					
					//{code: 'service_status', label: '<%=rb.getString("ZhuangTai")%>'},
					
					{code: 'ipsec_addr', label: '<%=rb.getString("IPSECDiZhi")%>'},
					
					{code: 'gps_position', label: '<%=rb.getString("GPSBanBen")%> + <%=rb.getString("GPSJingDu")%> + <%=rb.getString("GPSWeiDu")%> + <%=rb.getString("GPSGaoDu")%> '},
                    {code: 'electronic_downtilt', label: '<%=rb.getString("DianZiXiaQingJiao")%>'},
					//{code: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>'},
					//{code: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>'},
					//{code: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>'},
					//{code: 'install_address', label: '<%=rb.getString("AnZhuangXiangXiDiZhi")%>'},
					// {code: 'authCode',label:'<%=rb.getString("JianQuanMa")%>'},
				],
			
			othersCol= [
				{code: 'band', label: '<%=rb.getString("PinDuan")%>'},
				{code: 'sas_param', label: 'SAS <%=rb.getString("JianCeCanShuMing")%>'},
				{code: 'cell_neighbor', label: 'SAS <%=rb.getString("LinQu")%>'},
				{code: 'itfn_param', label: '<%=rb.getString("BeiXiangJieKou")%>'},
				{code: 'son_pci', label: 'SON PCI'},
				{code: 'rollback_enable', label: '<%=rb.getString("HuiTui")%> <%=rb.getString("SheZhiKaiGuan")%>'},
				{code: 'rollback_version', label: '<%=rb.getString("HuiTuiBanBen")%>'},
				{code: 'uboot_version', label: '<%=rb.getString("UBootVersion")%>'},
				{code: 'kernel_version', label: '<%=rb.getString("KernalVersion")%>'},
				{code: 'is_https', label: 'Https <%=rb.getString("ZhuangTai")%>'},
				{code: 'lan_enable', label: '<%=rb.getString("LANZhuangTai")%>'},
				{code: 'wan_ip', label: '<%=rb.getString("WanIPDiZhi")%>'},
				{code: 'lte_turbo_enable', label: 'LTE Turbo'},
				{code: 'eu_ru', label: 'EU/<%=rb.getString("RUShu")%>'},
				{code: 'halob_license', label: 'HaloB License'},
				{code: 'lock_mac_addr', label: '<%=rb.getString("SuoDingMac")%>'},
				{code: 'lock_mac_status', label: '<%=rb.getString("SuoDingMac")%> <%=rb.getString("ZhuangTai")%>'},
				{code: 'ipsec_bind_interface', label: 'IPSec Bind Interface'},
				{code: 'lgw_basic', label: 'WCG <%=rb.getString("JianCeCanShuMing")%>'},
				{code: 'lgw_mode_ue_speed_statistics', label: 'UE Speed Statistics'},
				{code: 'ipsec_auto_enroll', label: 'IPSec Auto Enroll'},
				{code: 'slot', label: 'Slot'},
			];

			return {
				alarmSync:true,
				basicCol: basicCol.filter(function(item){ return !ignores.includes(item.code);}),
				othersCol: othersCol.filter(function(item){ return !ignores.includes(item.code);}),
				form: {
					basic:[],
					others:[]
				},
				expanded: {
					params:true,
					basic:true,
					others:true,
				},
				monitorCols:{
					'serial_number':'serial_number',
					'mac_address':'MAC',
					'host_name':'cell_name',
					'op_state':'cell_status',
					'CELL_IDENTITY':'ECI',
					'PHYCELLID':'PCI',
					'plmnid':'plmn',
					'cell_ip':'IP',
					'IPSEC_ADDR':'ipsec_addr',
					'mmepool_ipsec_addr':'mme_addr',
					'EARFCNDLINUSE':'earfcn',
					'synStatus':'sync_status',
					'gps_satellite_count':'gps_satellites',
					'network_model':'duplex_mode',
					'gps_version':'gps_position',
					'gps_longitude':'gps_position',
					'gps_latitude':'gps_position',
					'gps_height':'gps_position',
					'signment':'sub_frame_assignment',
					'specialSubframe':'sub_frame_assignment',
					'rootIndex':'root_sequence_index',
					'validity':'lease',
					'lock_status':'lease'
				},
		
			};
		},
		computed: {
			isCloud() {
				return isCloud == 'true';
			},
			isSuperAdmin() {
				return is_super_user == 'true';
			},
			colSel() {
					
				var vm = this;
				
				//两组全选
				if(vm.colAll){
					return false
				}else if(vm.form.basic.length>0 || vm.form.others.length>0){ //半选 
					return true
				}else {
					return false;
				}
				
			},
			colAll() {
				var vm = this;
				return vm.basicAll && vm.othersAll;
			},
			basicAll() {
				var vm = this;
				return vm.basicCol.length == vm.form.basic.length;
			},
			othersAll() {
				var vm = this;
				return vm.othersCol.length == vm.form.others.length;
			},
			 
			showCols() {
				var vm = this;
				return vm.form.basic.concat(vm.form.others);
			}
		},
		methods: {
			init(){
				var showCols = enbvm.showProps;
				var vm = this;
				Object.assign(vm.form.basic,showCols);
				
				for(var key in vm.monitorCols){
					if(showCols.includes(key)){
						vm.form.basic.push(vm.monitorCols[key]);
						vm.form.basic.splice(vm.form.basic.indexOf(key),1);
					}
				}

			},
			comfirmSync() {
				var vm = this,
					url='',
					params = {
						smallCellCode: '',
						selectedParams: ''
					};
				
								
					var content ='';
					var selectArr = [];
					selectArr.push(vm.form.basic);
					selectArr.push(vm.form.others);
									
					for(var i=0;i<selectArr.length;i++){
						if(selectArr[i].length>0){
							content += ','+selectArr[i].join(',');
						}
					}

		
				if(vm.alarmSync) {
					content += ',sync_alarm';
				}
				content = content.substring(1);

				params.selectedParams =  content;

				if(enbvm.batchSync){
					params.smallCellCode = enbvm.batchCode;
					url="${ctx}/cell/quicksettings/batchSyncCell.action"
				}else {
					params.smallCellCode = enbvm.selectedRow.small_cell_code;
					url="${ctx}/cell/param/refreshCellInfo.action"
				}

				$.post(url, params, function(data){
					if (data["success"]) {
						vm.close();
						enbvm.clearSelection();
						enbvm.refreshList();
					} else {
						showMsg('error_msg',data["message"]);
					}
				}, "json");
				
			},
			close() {
				enbvm.openSyncDialog = false;
				enbvm.batchSync = false;
				enbvm.batchCode = '';
				enbvm.clearSelection();
			},
			
			colAllChange(val) {
				var vm = this;

				vm.basicAllChange(val);
				vm.othersAllChange(val);
				
			},
			basicAllChange(val){
				var vm = this,
					fields = vm.basicCol.map(function(item){
						return item.code;
					}),
					filters = vm.basicCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.basic = val?fields:filters;
			},
			othersAllChange(val){
				var vm = this,
					fields = vm.othersCol.map(function(item){
						return item.code;
					}),
					filters = vm.othersCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.others = val?fields:filters;
			},
			  
		},
		mounted() {
			this.init();
		}
	});

</script>
