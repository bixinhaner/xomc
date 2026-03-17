<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#gnbRollbackAddTask .el-form-item__label{
	line-height:26px;
	font-size: 12px;
}
#gnbRollbackAddTask .modeItem .el-radio{
	display:block;
	margin-left:0px;
	margin-bottom:20px;
} 
#gnbRollbackAddTask .modeItem{
	margin-top:20px;
}
#gnbRollbackAddTask .tableDiv{
	height:15px;
	width:auto;
}
#gnbRollbackAddTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#gnbRollbackAddTask .deviceItem{
	display:inline-block;
	margin-right:60px;
}
#gnbRollbackAddTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#gnbRollbackAddTask .titleStyML{
	margin-left: 20px;
}
#gnbRollbackAddTask .deviceTableBox{
	margin:10px 0px 0px 45px;
}
#gnbRollbackAddTask .gnbDeviceTableBox{
	width: 100%;
	font-size: 12px !important;
}
#gnbRollbackAddTask .gnbDeviceTableBox .pairgrid-right{
	top:40px!important;
	height: calc(100% - 40px)!important;
}
#gnbRollbackAddTask .gnbDeviceTableBox .el-pairgrid-title{
	top:15px!important;
	right: 15px!important;
}
#gnbRollbackAddTask .gnbDeviceTableBox .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
#gnbRollbackAddTask .editButton{
	position: absolute;
	right: 140px;
	top: 15px;
	padding:0 10px;
	height:24px;
	background:#F2F9FF;
	border-radius:2px;
	line-height:24px;
	cursor:pointer;
	margin-left:10px;
	border:1px solid #1DA3FC;
	display:inline-block;
}
#gnbRollbackAddTask .editButton i{
	font-size:14px !important;
}
#gnbRollbackAddTask .editButton span{
	font-size:12px;
}
.dialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
#gnbRollbackAddTask .pairgrid-right .el-ctable-toolbar{
	padding: 10px!important;
}
</style>

<!-- 新建gnb回退任务 -->
<div id="gnbRollbackAddTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:20px;' label-position="left"  :hide-required-asterisk=true>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' label-width="120px" prop='taskName' style='margin-left:45px;margin-top:20px;'>
			<el-input maxlength=100 :disabled='viewFlag' v-model="ruleForm.taskName" size="mini" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div>
			<div class="group-title not-extend titleStyML" >
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			</div>
			<el-form-item label-width="150px" style='margin:18px 0px 20px 45px;' label="<%=rb.getString("ChangPinXingHao")%>">
				<el-select v-model="ruleForm.productValue" @change="productChange">
					<el-option v-for="item in buttonGroups" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' label-width="150px" style="margin:20px 0px 0px 45px;">
				<el-radio-group  v-model="ruleForm.selectAll" :disabled="viewFlag" @change="selectAllChange" style="padding-top: 5px;">
					<el-radio  label="true"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<div class="deviceTableBox">
				<div class="gnbDeviceTableBox">
					<el-pairgrid 
						:id="'select_device_list'" 
						v-if="showPairGrid && ruleForm.selectAll == 'false'" 
						style="margin-right:45px;" query-name="serial_number" 
						:rownumber="true" 
						ref="gnbDevicePairgrid"  
						@selection-change='selectChange' 
						:right-url="rightUrl" :left-url="leftUrl" 
						:height="height" row-key="small_cell_code" 
						:query-params="queryParams" :title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}"
					 >
						<template slot="left">
							<el-table-column type="selection" width="45"></el-table-column>
							<el-table-column prop="connection_status" width="50">
								<template slot-scope="scope">
									<div :class="{
										'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
										'':scope.row.have_connected==2,
										'conn_exc':scope.row.connection_status=='Exception',
										'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
								</template>
							</el-table-column>
							<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
							<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' min-width="180"></el-table-column>
							<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="80"></el-table-column>
							<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="180"></el-table-column>
						</template>
						<template slot='toolbar'>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
								<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/<%=rb.getString("IPDiZhi")%>'"
								:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template slot="form">
										<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
											<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
											<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
										</el-form-item>
                                        <el-form-item class='deviceItem' label='<%=rb.getString("IPDiZhi")%>' prop='cell_ip'>
                                            <el-input v-model='query_cell_form.cell_ip'  size="mini"></el-input>
                                        </el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
											<el-select v-model="query_cell_form.group_id" size="mini">
												<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HuiTuiBanBen")%>' prop='rollback_version'>
											<el-select v-model="query_cell_form.rollback_version" size="mini">
												<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("RuanJianBanBen")%>' prop='software_version'>
											<el-select v-model="query_cell_form.software_version" size="mini">
												<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
									</template>
								</el-query>
								<div class="editButton" v-show="false" size="mini" @click="addBatchSn">
									<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
									<span><%=rb.getString("PiLiangShuRu")%></span>
								</div>
							</el-form>
						</template>
						<template slot='right'>
							<el-table-column prop="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="150"></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						:id="'all_device_list'" 
						v-if="ruleForm.selectAll == 'true'" 
						:query-params="queryParams" 
						style="border:1px solid #E9E9E9;margin-right:45px;" 
						ref="gnbDevicePairgrid"
						:url="leftUrl" 
						:height="height" 
						pagination="true" 
						>
						<template slot='toolbar'>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/<%=rb.getString("IPDiZhi")%>'"
								:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template slot="form">
										<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
											<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
											<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
										</el-form-item>
                                        <el-form-item class='deviceItem' label='<%=rb.getString("IPDiZhi")%>' prop='cell_ip'>
                                            <el-input v-model='query_cell_form.cell_ip'  size="mini"></el-input>
                                        </el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
											<el-select v-model="query_cell_form.group_id" size="mini">
												<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HuiTuiBanBen")%>' prop='rollback_version'>
											<el-select v-model="query_cell_form.rollback_version" size="mini">
												<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("RuanJianBanBen")%>' prop='software_version'>
											<el-select v-model="query_cell_form.software_version" size="mini">
												<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
									</template>
								</el-query>
							</el-form>
						</template>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
						<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' min-width="180"></el-table-column>
						<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="80"></el-table-column>
						<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
						<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' min-width="180"></el-table-column>
						<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' min-width="180"></el-table-column>
						<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="180"></el-table-column>
					</el-ctable>
					<div v-if="!showPairGrid && ruleForm.selectAll == 'false'" >
						<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="queryParams">
							<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
							<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' min-width="180"></el-table-column>
							<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="80"></el-table-column>
							<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="180"></el-table-column>
						</el-ctable>
					</div>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-left:45px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item  prop='status'>
				<el-radio-group v-model="ruleForm.status" style='margin-top:17px;' :disabled='viewFlag'>
					<el-radio label="active" style='margin-right:110px;margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
					<el-radio label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;' >
				<el-date-picker style='margin-top:10px;vertical-align:middle;' value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="viewFlag||setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
	<el-dialog class='dialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
new Vue({
	el:'#gnbRollbackAddTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/upgrade/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskName.trim(),
                    isGnb:1,
					taskType:vm.ruleForm.taskType
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("RenWuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.status !== 'timing'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		};
		var validateCellCodes = (rule,value,callback) => {
			if(this.ruleForm.selectAll == 'true'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		};
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value,
				snArr =  serialNumber.split(/[(\r\n)\r\n]+/g),
				list = [];
			
			if(snArr.length>1) {// 多行
				snArr.map(function(str){
					var item = str.trim(),
						lastIdx = item.lastIndexOf(';'),
						length = item.length-1;
					
					if(lastIdx>=0 && lastIdx == length) {
						list.push(item.substring(0,lastIdx));
					}else if(item) {
						list.push(item);
					}
				});
			}else {// 单行
				list = serialNumber.split(';');
			}
			
			//判断最后一项是否为空 为空删除
			if(list[list.length-1] == ""){
				list.splice(list.length-1)
			}
			
			var temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
		    
			if (serialNumber == null || serialNumber.length == 0) {
				callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
			}else {
				var nameFlag = list.every(function(item,index){
					return temp.test(item)
				})
				if(nameFlag){
					callback()
				}else{
					callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
				}
			}
		};
		return {
			deviceData:{}, // 新建任务接收的设备信息
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			selection:'',
			ruleForm:{
				taskName:'${addTaskName}',
				taskType:2,
				cellCodes:'',
				status:'active',
				exetime:'',
				selectAll:'false',
				version:'',
				productValue:'',
			},
			rules:{
				taskName:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{validator:validateCellCodes,trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			queryParams:{
				search_text:'',
				timeZone:timeZone,
				isGnb:1,
				like_fields: 'serial_number,host_name,cell_ip',
				group_id:'',
				serial_number:'',
				host_name:'',
				productValue:'',
				software_version:'',
				cell_ip:'',
				rollback_version:"",
			},
			query_cell_form:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				cell_ip:'',
				rollback_version:""
			},
			groupOptions:[],
			versionOptions:[],
			rbVersionOptions:[],
			taskId:'',
			operationType:'',
			defaultTaskName:'',
			showPairGrid:true,
			viewFlag:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.getNow()-8.64e7;
				}
			},
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:'<%=rb.getString("TianJia")%>',
			buttonGroups:[],
			clearTableFlag:false,
		}
	},
	methods:{ 
		/**
		 * 初始化
		 * @param id:number     gnb升级任务id
		 * @param type:string   任务类型  'addTask' -- 新增  'viewTask'-- 查看  'modifyTask' -- 修改
		*/
		init(id,type){ 
			var vm = this;
			vm.taskId = id ;
			vm.operationType = type;
			
			axios.post("${ctx}/task/upgrade/getProductType.action?isGnb=1").then(function(res){
            	var data = res.data;
				
		   		vm.buttonGroups = data;
			});
			
			if(type == 'viewTask'){
				vm.viewFlag = true;
				vm.showPairGrid = false;
			}else if(type == 'addTask'){
				vm.ruleForm.productValue = gnbFileVue.productValue;
				if(gnbFileVue.cellData.length > 0){
					vm.$refs.gnbDevicePairgrid.appendCheckedRows(gnbFileVue.cellData);
				}
			}
			vm.leftUrl= '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
			setTimeout(function(){
				vm.commonSelection();
			},500)
			if(type !== "addTask"){
				vm.$nextTick(function(){
					vm.rightUrl='${ctx}/task/upgrade/getTaskSelectedList.action?taskId=' +vm.taskId;
				})
				vm.getTaskDateInfo();
			}
			initForm(vm.$refs.ruleForm);
		},
		// 设备执行类别  1 全部执行 2 指定执行
		selectAllChange(val){
			var vm = this;

			vm.resetQuery();
			if(val == 'true'){
				vm.rightUrl = '';
			}else{
				vm.rightUrl='${ctx}/task/upgrade/getTaskSelectedList.action?taskId='+vm.taskId;
			}
			
		},
		commonSelection(){
			var vm = this;
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data;
			}).catch(function(error){})			
		},
		// 模糊查询
		query(val){
			var vm = this;
			vm.queryParams.search_text = val;
		},
		// 高级查询
		advanceQuery(){
			var vm = this;
			vm.queryParams.search_text = "";
			Object.assign(vm.queryParams,vm.query_cell_form);
		},
		// 重置
		resetQuery(){
			var vm = this,
				params = {
					group_id:'',
					serial_number:'',
					host_name:'',
					software_version:'',
					cell_ip:'',
					rollback_version:''
				};
			Object.assign(vm.query_cell_form,params);
			Object.assign(vm.queryParams,params);
		},
		// 批量输入
		addBatchSn(){
			var vm = this;
			vm.listVisible = true;
		},
		//取消批量输入
		closeBatchSn(){
			var vm = this;
			vm.listVisible = false;
			vm.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '';
			snStr = snStr.replace(/[(;\s*)(\r\n)\r\n]+/g,';');
			
			var params = {
					serialNumbers : snStr,
					productType : vm.ruleForm.productValue
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/cpeinfos/getTaskCheckSNList.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){		
							vm.$refs.gnbDevicePairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
						}
					})
				}
			})
		},
		// gnb设备选择事件
		selectChange(selection){
			var vm = this;
			vm.selection = selection;
		},
		// 获取gnb升级任务详情
		getTaskDateInfo(){
			var vm = this,
				params={
					taskId:vm.taskId,
					timeZone:timeZone
				};
			
			axios.post('${ctx}/task/upgrade/getTask.action',stringify(params)).then(function(response){
				let data = response.data;
				vm.ruleForm.taskName = data.TASK_NAME;
				vm.ruleForm.status = data.CREATE_STATUS;
                vm.ruleForm.productValue = data.PRODUCT_TYPE;
				if(data.CREATE_STATUS == 'timing'){
					vm.ruleForm.exetime = data.CREATE_TIME;
				}else{
					vm.ruleForm.exetime = '';
				}
				vm.defaultTaskName = data.TASK_NAME;
				vm.ruleForm.selectAll = data.selectAll;
				if(data.selectAll == ''){
					vm.ruleForm.selectAll = 'false';
				}
				
			}).catch(function(error){})
		},
		// 提交
		submit(){
	    	var vm = this,
                message = '<%=rb.getString("ChengGong")%>';
            // 防止多次提交
			if(gnbFileVue.slideSubmitLoading)return

			if(vm.operationType == 'modifyTask'){
				if(!isFormChanged(vm.$refs.ruleForm)){
					vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						type:'warning'
					}).then().catch();
					return;
				}
			}
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},urls='';
	    			params.timeZone = timeZone;
					params.isGnb = 1;
	    			params.cellCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskName;
					params.selectAll = vm.ruleForm.selectAll;
	    			params.status = vm.ruleForm.status;
					params.taskType = vm.ruleForm.taskType;
                    params.productValue = vm.ruleForm.productValue;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
					if(vm.operationType == 'addTask'){
						urls = '${ctx}/task/upgrade/addTask.action';
					}else{
						params.taskId = vm.taskId
						urls = '${ctx}/task/upgrade/updateTask.action';
					}
					vm.saveTask(urls,params,message);
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(urls,params,message){
			var vm = this;
            gnbFileVue.slideSubmitLoading = true;
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    gnbFileVue.$refs.upgrade_cell_table.clearSelection();
                    gnbFileVue.$refs.slide.hide();
                    gnbFileVue.list_name = "software";
                    isJumpToPage = '';
				}else{
					vm.$message.error(data["message"]);
                    gnbFileVue.slideSubmitLoading = false;
				}
			}).catch(function(error){})
			
		},
		// 取消
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
			if(vm.operationType !== 'taskView'){
				if(isFormChanged(this.$refs.ruleForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						gnbFileVue.$refs.slide.hide();
						isJumpToPage = ''
					}).catch(() => {
						
					})
				}else{
					gnbFileVue.$refs.slide.hide();
					isJumpToPage = ''
				}
			}else{
				gnbFileVue.$refs.slide.hide();
				isJumpToPage = ''
			}
			
		},
		// 定时时间失焦事件
		setTime(){
			var vm = this;
			vm.ruleForm.exetime = formatDate(new Date(gloableTime));
			vm.$refs.ruleForm.validateField('exetime');
		},
		// 产品类型改变
		productChange(val){
			var vm = this;
			vm.commonSelection();
		},
	},
	watch:{
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		selection(){
			var vm = this,
				data = this.$refs.gnbDevicePairgrid.getData();
				cellCodes = '',cellCodeList = [];
			if(data.length != 0){
				data.map(function(item){
					cellCodeList.push(item.small_cell_code);
				})
			}
			cellCodes = cellCodeList.join(',');
			vm.ruleForm.cellCodes = cellCodes;
		},
		"ruleForm.productValue":function(newVal){
			this.queryParams.productValue = newVal;
			if(this.clearTableFlag == true){
				this.$refs.gnbDevicePairgrid.clear();
			}
			this.clearTableFlag = true;
		},
	},
	mounted(){
		eventBus.$off('task-init').$on('task-init',this.init);
		eventBus.$off('add-task').$on('add-task',this.submit);
		eventBus.$off('cancel-add-task').$on('cancel-add-task',this.cancel);
	}
})
</script>