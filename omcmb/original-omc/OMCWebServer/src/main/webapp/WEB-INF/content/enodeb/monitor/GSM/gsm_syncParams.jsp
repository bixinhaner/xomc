<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
	#gsm_syncParams .columns-list-cls {
		max-width: 920px;
	}
	#gsm_syncParams .columns-list-cls .el-checkbox {
		min-width: 210px;
	}
	#gsm_syncParams .el-checkbox__label {
		font-size: 12px;
		font-weight: normal;
	}
	#gsm_syncParams .select-all-cls {
		padding: 10px 0 0 9px;
		display: flex;
		align-items: center;
	}
	#gsm_syncParams .select-all-cls > span {
		font-weight: bold;
		margin-left: 10px;
	}
	#gsm_syncParams .select-all-cls > i {
		margin-right: 5px;
	}
	#gsm_syncParams .gray-color {
		font-size: 12px;
		color: #999;
	}
	#gsm_syncParams .el-icon-close1::before {
		color: #333;
	}
</style>
<div id="gsm_syncParams">
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
					<i :class="{'el-icon':true,'el-icon-close1':expanded.bsc,'el-icon-open':!expanded.bsc}" @click="expanded.bsc = !expanded.bsc"></i>
					<el-checkbox :indeterminate="form.bsc.length < bscCol.length && form.bsc.length>0" v-model="bscAll" @change="bscAllChange"></el-checkbox> 
					<span>BSC</span>
				</div>
				<el-checkbox-group v-show="expanded.bsc" class="col-group" v-model="form.bsc">
					<el-checkbox v-for="item in bscCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
				</el-checkbox-group>
			</div>
            <div>
				<div class="select-all-cls">
					<i :class="{'el-icon':true,'el-icon-close1':expanded.bts,'el-icon-open':!expanded.bts}" @click="expanded.bts = !expanded.bts"></i>
					<el-checkbox :indeterminate="form.bts.length < btsCol.length && form.bts.length>0" v-model="btsAll" @change="btsAllChange"></el-checkbox> 
					<span>BTS</span>
				</div>
				<el-checkbox-group v-show="expanded.bts" class="col-group" v-model="form.bts">
					<el-checkbox v-for="item in btsCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
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
	var gsmSyncParamVue = new Vue({
		el: '#gsm_syncParams',
		data() {
			var 
				basicCol = [
					{code: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>'},
					{code: 'software_version', label: '<%=rb.getString("SoftwareVersion")%>'},
					{code: 'firmware_version', label: '<%=rb.getString("FirmwareVersion")%>'},
					{code: 'MAC', label: '<%=rb.getString("XiaoZhanMAC")%>'},
					{code: 'IP', label: '<%=rb.getString("IPDiZhi")%>'},
					{code: 'ue_count', label: '<%=rb.getString("UEShu")%>'},
                    {code: 'halob_license', label: 'License'},
					// {code: 'authCode', label: '<%=rb.getString("JianQuanMa")%>'},
				],
                bscCol = [
                    {code: 'BtsNum', label: '<%=rb.getString("BTSShu")%>'},
                ],
                btsCol = [
                    {code: 'cell_status', label: '<%=rb.getString("ShiFouJiHuo")%>'},
					{code: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>'},
                    {code: 'sync_status', label: '<%=rb.getString("TongBuZhuangTai")%>'},
                    {code: 'gps_satellites', label: '<%=rb.getString("GPSWeiXingShu")%>'},
                    {code: 'currentLac', label: 'LAC'},
                    {code: 'currentArfcn', label: '<%=rb.getString("PinDian")%> + <%=rb.getString("ShangXingPinLv")%> + <%=rb.getString("XiaXingPinLv")%>'},
                    {code: 'gps_position', label: '<%=rb.getString("GPSJingDu")%> + <%=rb.getString("GPSWeiDu")%> + <%=rb.getString("GPSGaoDu")%> '},
                    {code: 'bts_bsc_relationship', label: 'Ipa Unit Id + Oml Remote Ip + Oml Remote Ip Bak + BSC Select + <%=rb.getString("BSCLianJieZhuangTai")%> + <%=rb.getString("SuoShuBSCBianMa")%>'},
                ];

			return {
				alarmSync:true,
				basicCol: basicCol,
                bscCol: bscCol,
                btsCol: btsCol,
				form: {
					basic:[],
                    bsc:[],
                    bts:[]
				},
				expanded: {
					params:true,
					basic:true,
                    bsc:true,
                    bts:true
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
				}else if(vm.form.basic.length>0 || vm.form.bsc.length>0 || vm.form.bts.length>0){ //半选 
					return true
				}else {
					return false;
				}
				
			},
			colAll() {
				var vm = this;
				return vm.basicAll && vm.bscAll && vm.btsAll;
			},
            basicAll() {
				var vm = this;
				return vm.basicCol.length == vm.form.basic.length;
			},
            bscAll() {
				var vm = this;
				return vm.bscCol.length == vm.form.bsc.length;
			},
            btsAll() {
				var vm = this;
				return vm.btsCol.length == vm.form.bts.length;
			},
			showCols() {
				var vm = this;
				return vm.form.basic.concat(vm.form.bsc).concat(vm.form.bts);
			}
		},
		methods: {
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
                    selectArr.push(vm.form.bsc);
                    selectArr.push(vm.form.bts);
									
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

				if(gsmvm.batchSync){
					params.smallCellCode = gsmvm.batchCode;
					url="${ctx}/cell/quicksettings/batchSyncCell.action"
				}else {
					params.smallCellCode = gsmvm.selectedRow.small_cell_code;
					url="${ctx}/cell/param/refreshCellInfo.action"
				}

				$.post(url, params, function(data){
					if (data["success"]) {
						vm.close();
						gsmvm.clearSelection();
						gsmvm.refreshList();
					} else {
						showMsg('error_msg',data["message"]);
					}
				}, "json");
				
			},
			close() {
				gsmvm.openSyncDialogShow = false;
				gsmvm.batchSync = false;
				gsmvm.batchCode = '';
				gsmvm.clearSelection();
			},
			
			colAllChange(val) {
				var vm = this;

				vm.basicAllChange(val);
                vm.bscAllChange(val);
                vm.btsAllChange(val);
				
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
            bscAllChange(val){
				var vm = this,
					fields = vm.bscCol.map(function(item){
						return item.code;
					}),
					filters = vm.bscCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.bsc = val?fields:filters;
			},
            btsAllChange(val){
				var vm = this,
					fields = vm.btsCol.map(function(item){
						return item.code;
					}),
					filters = vm.btsCol.filter(function(item){
						return item.disabled;
					}).map(function(item){
						return item.code;
					});
				
				vm.form.bts = val?fields:filters;
			},
		},
		mounted() {}
	});

</script>
