<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#sasLogPage{
		background-color: #F6F7F8;
		width: 100%;
	}
	#sasLogPage .logHeader{
		background-color: #FFFFFF;
		margin-bottom: 10px;
	}
	#sasLogPage .headTitleBox{
		height: 50px;
		width: 100%;
		position: relative;
		border-bottom: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
		font-size: 16px;
		font-weight: bold;
		padding-left: 20px;
		box-sizing: border-box;
	}
	#sasLogPage .headTimeQueryBox{
		height: 60px;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	#sasLogPage .headTimeQueryBox .el-input__icon::before{
		color: #4D84FF;
	}
	#sasLogPage .headTimeQueryBox .el-date-editor .el-range-separator{
		line-height: unset;
	}
	#sasLogPage .logContent{
		display: flex;
		height: calc(100% - 130px) !important;
		margin-top: 20px;
		width: 100%;
	}
	#sasLogPage .logContent .contentLeftBoxCls{
		flex: 1;
		width: 0;
		margin: 0px 10px;
		height: 100%;
		border:1px solid #E9E9E9;
		border-top: 2px solid #4D84FF; 
		box-sizing:border-box; 
		background-color: #FFFFFF;
	}
	#sasLogPage .logContent .contentRightBoxCls{
		flex: 1;
		width: 0;
		margin: 0px 10px;
		height: 100%;
		border:1px solid #E9E9E9;
		border-top: 2px solid #4D84FF; 
		box-sizing:border-box; 
		background-color: #FFFFFF;
	}
	#sasLogPage .logContent .tableTitleCls{
		position: relative;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: flex-end;
		font-size: 12px;
		color: #333333;
		font-weight: bold;
		padding: 0 10px;
		
	}
	#sasLogPage .tableTitleCls .el-icon::before{
		font-size: 18px;
	}
	#sasLogPage .tableTitleCls .tableTitleNameCls{
		position: absolute;
		top: 50%;
		left:50%;
		transform: translate(-50%,-50%);
	}
	#sasLogPage .logContent .tableBoxCls{
		height: calc(100% - 40px) !important;
	}
	.activeStatusItem .el-icon,.inactiveStatusItem .el-icon{
		font-size:20px;
		vertical-align:bottom;
		margin-right:5px;
	}
	.activeStatusItem .el-icon-status-active:before{
		color:#67D972;
	}
	.inactiveStatusItem .el-icon-status-active:before{
		color:#E88282;
	}
	.sasMsgBoxCls .el-icon::before{
		font-size: 12px;
	}
	.queryTimeCls .el-button--text{
		display: none;
	}
	.msgTipBoxCls{
		white-space:pre-wrap;
		max-height:300px;
		max-width: 300px;
		word-break: break-all;
		word-wrap: break-word;
		font-size: 12px;
		overflow: auto;
		padding: 5px;
		box-shadow: 2px 2px 5px #E9E9E9;
	}
	#sasLogPage .cellBoxCls{
		position: absolute;
		height: 24px;
		left: 50%;
		top: 50%;
		transform: translate(-50%,-50%);
	}
	#sasLogPage .selectBox{
		display: flex;
		margin-right: 5%;
		font-weight: 500;
		font-size: 12px;
		justify-content: center;
		margin-bottom: 30px;
		height: 30px;
	}
	#sasLogPage .selectBox div{
		width:60px;
		height: 24px;
		line-height: 24px;
		border-radius: 2px 0px 0px 2px;
		border:#e9e9e9 solid 1px;
		text-align: center;
		cursor: pointer
	}
	#sasLogPage .selectBox .btnSelectCls{
		border:#4D84FF solid 1px ;
		color: #4D84FF
	}
	#sasLogPage .sasMainLogBoxCls{
		height: calc(100% - 60px) !important;
		width: 100%;
	}
	.sasMsgPopoverCls textarea {
		width: 100%;
		height: 100%;
		resize: none;
		background-color: #FFFFFF;
	}
</style>
<div class="panelDefault" id="sasLogPage" style="overflow:hidden">
	<div class="logHeader">
		<div class="headTitleBox">
			<%=rb.getString("Log")%>
			<span v-if="sasLogType == 'single'" style="color:#4D84FF;margin-left:10px;">(SN:{{serialNumber}})</span>
			<div v-if="sasLogType == 'single'" class="cellBoxCls">
				<div class="selectBox" v-if="isTC">
					<div :class="{btnSelectCls:carrierType == '1'}" @click="selectTabClick('1')">Cell 1</div>
					<div :class="{btnSelectCls:carrierType == '2'}" @click="selectTabClick('2')">Cell 2</div>
					<div :class="{btnSelectCls:carrierType == '3'}" @click="selectTabClick('3')">Cell 3</div>
				</div>
			</div>
			<div class="newIconBoxCls-bt" style="right:20px;top:12px;" @click="closeSasLog" tip="<%=rb.getString("GuanBi")%>">
				<span class="el-icon el-icon-close"></span>
			</div>
		</div>
		<!--@change="sasLogTimeChange"  -->
		<div class="headTimeQueryBox" v-if="sasLogType == 'single'">
			<el-date-picker 
				size="small" 
				popper-class="queryTimeCls"
				:clearable="false"
				v-model="sasLogTime" 
				:picker-options="pickerOptions"
				value-format="yyyy-MM-dd HH:mm:ss" 
				type="datetimerange" 
				@change="sasLogTimeChange"
				range-separator="——"  
				start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
				end-placeholder='<%=rb.getString("JieShuShiJian")%>'
			></el-date-picker>
		</div>
	</div>
	<div class="logContent" v-if="sasLogType == 'single'">
		<div class="contentLeftBoxCls" v-show="sasLogPackType == 'shrink' || (sasLogPackType == 'expand' && activePack == 'main')">
			<div class="tableTitleCls">
				<div class="tableTitleNameCls">Message Logs</div>
				<div>
					<span class="el-icon el-icon-operation-export" @click="exportMessageLogTable"></span>
					<span v-show="optBtnShow" style="margin:0px 10px;" class="el-icon el-icon-operation-delete" @click="sasLogDel('main')"></span>	
					<span v-show="sasLogPackType == 'shrink'" class="el-icon el-icon-fullscreen" @click="sasLogPackChange('main','expand')"></span>
					<span v-show="sasLogPackType == 'expand'" class="el-icon el-icon-fullscreen-exit" @click="sasLogPackChange('main','shrink')"></span>
				</div>
			</div>
			<!--:url="sasMessageLogTableUrl" :data="sasMessageLogTableData"-->
			<div class="tableBoxCls">
				<el-ctable 
					ref="sasMessageLogTable"
					:rownumber="true" 
					id="sasMessageLogTable" 
					:url="sasMessageLogTableUrl"
					:query-params="queryParamsMessageLog"
					height="100%" 
					pagination="true"
					@sort-change="sortChangeMessageLog"
				>   
					<el-table-column label='<%=rb.getString("SASFangXiang")%>' min-width="70" prop="from"></el-table-column>
					<el-table-column label='<%=rb.getString("MuBiao")%>' min-width="70" prop="to" show-overflow-tooltip></el-table-column>
					<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' min-width="120" prop="logName" show-overflow-tooltip></el-table-column>
					<el-table-column label='<%=rb.getString("XiaoXi")%>' min-width="300" prop="msg" :show-overflow-tooltip="false">
						<template slot-scope="scope">
							<div class="sasMsgBoxCls">
								<el-popover placement="bottom" width="300" trigger="hover" popper-class="sasMsgPopoverCls">
									<textarea cols="50" rows="20" type="textarea" readonly>{{scope.row.msg}}</textarea>
									<div slot="reference" style="overflow: hidden;text-overflow: ellipsis;white-space: nowrap;width:100%;">{{scope.row.msg}}</div>
								</el-popover>
							</div>
						</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("ShiJian")%> (UTC)' min-width="120" prop="time" show-overflow-tooltip sortable></el-table-column>
				</el-ctable>
			</div>
			
		</div>
		<div class="contentRightBoxCls" v-show="sasLogPackType == 'shrink' || (sasLogPackType == 'expand' && activePack == 'cbsd')">
			<div class="tableTitleCls">
				<div class="tableTitleNameCls">Logs</div>
				<div>
					<span class="el-icon el-icon-operation-export" @click="exportSasSingleLogTable"></span>
					<span v-show="optBtnShow" style="margin:0px 10px;" class="el-icon el-icon-operation-delete" @click="sasLogDel('cbsd')"></span>	
					<span v-show="sasLogPackType == 'shrink'" class="el-icon el-icon-fullscreen" @click="sasLogPackChange('cbsd','expand')"></span>
					<span v-show="sasLogPackType == 'expand'" class="el-icon el-icon-fullscreen-exit" @click="sasLogPackChange('cbsd','shrink')"></span>
				</div>
				
			</div>
			<div class="tableBoxCls">
				<el-ctable 
					ref="sasSingleLogTable" 
					:rownumber="true" 
					id="sasSingleLogTable" 
					:url="sasSingleLogTableUrl"
					:query-params="queryParamsSingleLog" 
					height="100%"
					pagination="true"
					@sort-change="sortChangeSingleLog"
				>
					<el-table-column label="<%=rb.getString("CBSDID")%>" prop="cbsdId" width="140"></el-table-column>
					<el-table-column label="<%=rb.getString("GRANTID")%>" prop="grantId" width="140"></el-table-column>
					<el-table-column label="<%=rb.getString("SASZhuangTai")%>" prop="state" sortable width="140">
						<template slot-scope="scope">
							<div>
								<span v-if="scope.row.cpeSasStatusEqualOmc == '0' && scope.row.deviceType === 'CPE'" class="el-icon el-icon-circle-warning" ></span>
								<span v-if="scope.row.state == '0'">Unregistered</span>
								<span v-if="scope.row.state == '1'">Registered</span>
								<span v-if="scope.row.state == '3'">Granted</span>
								<span v-if="scope.row.state == '4'">Grant Suspended</span>
								<span v-if="scope.row.state == '5'">Authorized</span>
								<span v-if="scope.row.state == '6'">Transmission</span>
							</div>
						</template>
					</el-table-column>
					<el-table-column label="<%=rb.getString("ShiJian")%>(UTC)" prop="time" sortable width="160"></el-table-column>
					<el-table-column label="<%=rb.getString("XinXi")%>" prop="message" show-overflow-tooltip min-width="200"></el-table-column>
				</el-ctable>
			</div>
		</div>
	</div>
	<div class="sasMainLogBoxCls" v-if="sasLogType == 'all'">
		<el-ctable 
			ref="sasMainLogTable"
			:rownumber="true" 
			id="sasMainLogTable" 
			:url="sasMainLogTableUrl"
			:query-params="queryParamsMainLog"
			height="100%" 
			pagination="true"
			@sort-change="sortChangeMainLog"
		>
			<template slot="toolbar">
				<!-- 导出 -->
				<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
					<div class="newIconBoxCls-bt" style="right:60px;top:12px;" @click="refreshTable" tip="<%=rb.getString("ShuaXin")%>">
						<span class="el-icon el-icon-common-refresh"></span>
					</div>
					<div class="newIconBoxCls-bt" style="right:20px;top:12px;" @click="exportMainLog" tip="<%=rb.getString("DaoChu")%>">
						<span class="el-icon el-icon-operation-export"></span>
					</div>
					<el-query type="normal" @query="queryMainLog" placeholder="<%=rb.getString("Title_SheBeiBianMa")%>"></el-query>
				</div>
			</template>
            <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="230" prop="cbsd"></el-table-column>
			<el-table-column label='<%=rb.getString("SASFangXiang")%>' min-width="70" prop="from"></el-table-column>
			<el-table-column label='<%=rb.getString("MuBiao")%>' min-width="70" prop="to" show-overflow-tooltip></el-table-column>
			<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' min-width="120" prop="logName" show-overflow-tooltip></el-table-column>
			<el-table-column label='<%=rb.getString("XiaoXi")%>' min-width="300" prop="msg" :show-overflow-tooltip="false">
				<template slot-scope="scope">
					<div class="sasMsgBoxCls">
						<el-popover placement="bottom" width="300" trigger="hover" popper-class="sasMsgPopoverCls">
							<textarea cols="50" rows="20" type="textarea" readonly>{{scope.row.msg}}</textarea>
							<div slot="reference" style="overflow: hidden;text-overflow: ellipsis;white-space: nowrap;width:100%;">{{scope.row.msg}}</div>
						</el-popover>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShiJian")%> (UTC)' min-width="120" prop="time" show-overflow-tooltip sortable></el-table-column>
		</el-ctable>
	</div>
</div>

<script type="text/javascript">
	
	var sasLogPageVue = new Vue({
		el: '#sasLogPage',
		data(){
			var vm = this;
			return {
				serialNumber:'',
				sasLogTime:[],
				queryParamsMessageLog:{
					cbsd:'',
					startTime:'',
					endTime:''
				},
				sortMessageLog:'',
				orderMessageLog:'',
				sasMessageLogTableData:[],
				queryParamsSingleLog:{
					serialNumber: '',
					dualCarrierType: '',
					startTime:'',
					endTime:'',
				},
				sortSingleLog:'',
				orderSingleLog:'',
				carrierType:'',
				isTC:false,
				sasMessageLogTableUrl:'',
				sasSingleLogTableUrl:'',
				pickerMinDate:'',
				pickerOptions:{
					onPick:obj =>{
						this.pickerMinDate = new Date(obj.minDate).getTime()
					},
					disabledDate: time =>{
						if(this.pickerMinDate){
							const day1 = 7 * 24 * 3600 * 1000 - 1
							const maxTime = this.pickerMinDate + day1
							const minTime = this.pickerMinDate - day1
							return time.getTime() > maxTime || time.getTime() < minTime
						}
					}
				},
				activePack:'main',
				sasLogPackType:'shrink',
				sasMainLogTableUrl:'',
				queryParamsMainLog:{
					cbsd:'',
					startTime:'',
					endTime:''
				},
				sasLogType:''
			}
		},
		computed: {
			optBtnShow() {
				return writableMap['CODE_ADVANCE_SAS'] == true;
			},
		},
		watch: {
			'sasLogTime':function(newVal){
				if(newVal){
					this.pickerMinDate = ''
				}
			},
		},
		methods: {
			// 初始化
			init(row,type){
				var vm = this;
				vm.sasLogType = type;
				var dt = new Date();
				dt.setMinutes(dt.getMinutes()+dt.getTimezoneOffset());
				var nowTime = new Date(dt.getTime());
				var startTime = formatDate(addDate(nowTime,-6)).slice(0,10) + ' 00:00:00',
					endTime = formatDate(nowTime).slice(0,10) + ' 23:59:59';
				if(type == 'single'){
					vm.serialNumber = row.serialNumber;
					vm.queryParamsMessageLog.cbsd = vm.serialNumber;
					vm.queryParamsMessageLog.queryType = 'singleDevice';
					vm.queryParamsSingleLog.serialNumber = vm.serialNumber;

					if(row.cellType && row.cellType == 'TC'){
						vm.isTC = true;
						vm.carrierType = '1';
						vm.queryParamsMessageLog.cellSequenceId = '1';
						vm.queryParamsSingleLog.cellSequenceId = '1';
					}
					vm.queryParamsMessageLog.startTime = startTime;
					vm.queryParamsMessageLog.endTime = endTime;
					vm.queryParamsSingleLog.startTime = startTime;
					vm.queryParamsSingleLog.endTime = endTime;
					vm.sasLogTime = [startTime,endTime];
					vm.sasMessageLogTableUrl='${ctx}/cell/SAS/getMainLog.action';
					vm.sasSingleLogTableUrl='${ctx}/cell/SAS/getSasCbsdLog.action';
				}else{
					vm.sasMainLogTableUrl='${ctx}/cell/SAS/getMainLog.action';
				}
					
			},
			// 导出CBSD LOG表格
			exportMessageLogTable(){
				var vm = this,
					params = {
						timeZone: timeZone,
						cbsd: vm.serialNumber,
						startTime: vm.queryParamsMessageLog.startTime,
						endTime: vm.queryParamsMessageLog.endTime,
						sort: vm.sortMessageLog?vm.sortMessageLog:'',
						order: vm.orderMessageLog?vm.orderMessageLog:''
					};
				if(vm.isTC){
					params.cellSequenceId = vm.carrierType;
				}
				if(vm.sasLogType == 'single'){
					params.queryType = 'singleDevice';
				}
				exportByForm("${ctx}/cell/SAS/exportMainLog.action",params)
			},
			// 导出 SAS log
			exportSasSingleLogTable(){
				var vm = this,
					params = {
						timeZone: timeZone,
						serialNumber: vm.serialNumber,
						startTime:vm.queryParamsSingleLog.startTime,
						endTime:vm.queryParamsSingleLog.endTime,
						sort: vm.sortSingleLog?vm.sortSingleLog:'',
						order: vm.orderSingleLog?vm.orderSingleLog:''
					};
				if(vm.isTC){
					params.cellSequenceId = vm.carrierType;
				}
				exportByForm("${ctx}/cell/SAS/exportSasCbsdLog.action",params);
			},
			// // 时间范围改变事件
			sasLogTimeChange(val){
				var vm = this;
				if(val != null || val != undefined ){
					vm.queryParamsMessageLog.startTime = val[0];
					vm.queryParamsMessageLog.endTime = val[1];
					vm.queryParamsSingleLog.startTime = val[0];
					vm.queryParamsSingleLog.endTime = val[1];
				}

			},
			// 日志清空
			sasLogDel(type){
				var vm = this,
					urls = '',
					params={
						serialNumber: vm.serialNumber,
						timeZone: timeZone,
					},
					codes={
						'cbsd':'sasSingleLogTable',
						'main':'sasMessageLogTable'
					};
				
				if(type == 'cbsd'){
					urls = '${ctx}/cell/SAS/clearSasCbsdLog.action';
				}else{
					urls = '${ctx}/cell/SAS/clearMessageLog.action';
				}
				if(vm.isTC){
					params.cellSequenceId = vm.carrierType;
				}
				vm.$confirm('<%=rb.getString("QueRenQingKongSuoYouRiZhi")%>','<%=rb.getString("QueRen")%>').then(function(){
					axios.post(urls,stringify(params)).then(function(response){
						var data = response.data;
						if(data) {
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
								vm.$refs[codes[type]].refresh()
							}else{
								vm.$message.error(data["message"])
							}
						}
					}).catch(function(error){})
				}).catch(() => {});
			},
			closeSasLog(){
				eventBus.$emit('close-dialog');
			},
			//排序点击  --- MAIN LOG
			sortChangeMessageLog(data){ 
				var vm =this,
					order={
						ascending:'asc',
						descending:'desc'
					};
				vm.sortMessageLog = data.prop;
				vm.orderMessageLog = order[data.order];		
			},
			//排序点击  --- SAS LOG 
			sortChangeSingleLog(data){ 
				var vm =this,
					order={
						ascending:'asc',
						descending:'desc'
					};
				vm.sortSingleLog = data.prop;
				vm.orderSingleLog = order[data.order];		
			},
			// 刷新
			refreshTable(){
				var vm = this;
				var dt = new Date();
				dt.setMinutes(dt.getMinutes()+dt.getTimezoneOffset());
				var nowTime = new Date(dt.getTime());
				var startTime = formatDate(addDate(nowTime,-6)).slice(0,10) + ' 00:00:00',
					endTime = formatDate(nowTime).slice(0,10) + ' 23:59:59';
				if(vm.sasLogType == 'single'){
					vm.queryParamsMessageLog.startTime = startTime;
					vm.queryParamsMessageLog.endTime = endTime;
					vm.queryParamsSingleLog.startTime = startTime;
					vm.queryParamsSingleLog.endTime = endTime;
					vm.sasLogTime = [startTime,endTime];
					vm.$refs.sasSingleLogTable.refresh();
					vm.$refs.sasMessageLogTable.refresh();
				}else{
					vm.$refs.sasMainLogTable.refresh();
				}
				
			},
			// mian log  cbsd log 展开和收起
			sasLogPackChange(activePack,type){
				var vm = this;

				vm.activePack = activePack;
				vm.sasLogPackType = type;
			},
			// 小区切换事件
			selectTabClick(val){
				var vm = this;
				vm.carrierType = val;
				if(vm.isTC){
					vm.queryParamsMessageLog.cellSequenceId = vm.carrierType;
					vm.queryParamsSingleLog.cellSequenceId = vm.carrierType;
				}
				vm.refreshTable();
			},
			// Sas 所有日志 模糊查询
			queryMainLog(val){
				var vm = this;
				vm.queryParamsMainLog.cbsd = val;
			},
			//排序点击  --- SAS 全部日志
			sortChangeMainLog(data){ 
				var vm =this,
					order={
						ascending:'asc',
						descending:'desc'
					};
				vm.sortMainLog = data.prop;
				vm.orderMainLog = order[data.order];		
			},
			// Sas 所有日志 导出
			exportMainLog(){
				var vm = this,
					params = {
						timeZone: timeZone,
						cbsd: vm.queryParamsMainLog.cbsd,
						startTime: '',
						endTime: '',
						sort: vm.sortMainLog?vm.sortMainLog:'',
						order: vm.orderMainLog?vm.orderMainLog:''
					};
				var dt = new Date();
				dt.setMinutes(dt.getMinutes()+dt.getTimezoneOffset());
				var nowTime = new Date(dt.getTime());
				var startTime = formatDate(addDate(nowTime,-6)).slice(0,10) + ' 00:00:00',
					endTime = formatDate(nowTime).slice(0,10) + ' 23:59:59';
				params.startTime = startTime;
				params.endTime = endTime;
				exportByForm("${ctx}/cell/SAS/exportMainLog.action",params)
			},
		},
		created(){},
		mounted(){
			eventBus.$off('sasLog-init').$on('sasLog-init',this.init);
		}
	});
</script>