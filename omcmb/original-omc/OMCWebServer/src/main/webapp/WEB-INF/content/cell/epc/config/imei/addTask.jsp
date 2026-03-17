<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#addIMEIConfigTask .el-form-item__label{
	line-height:26px;
	text-align:left;
}
#addIMEIConfigTask .modeItem{
	margin-top:20px;
}
#addIMEIConfigTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#addIMEIConfigTask .pairgrid-left,#addIMEIConfigTask .pairgrid-left .el-ctable{
	border:none;
	border-left:none;
}
.deviceSelectItem .el-radio__label{
	font-size:12px;
}
#addIMEIConfigTask .el-ctable-toolbar{
	position: relative;
}
#addIMEIConfigTask .editButton{
	position: absolute;
	right: 120px;
	padding:0 10px;
	height:24px;
	background:#F2F9FF;
	border-radius:2px;
	line-height:24px;
	cursor:pointer;
	margin-left:10px;
	border:1px solid #1DA3FC;
	display:inline-block;
	margin-top:-50px;
}
#addIMEIConfigTask .editButton i{
	font-size:14px !important;
}
#addIMEIConfigTask .editButton span{
	font-size:12px;
}
.addDeviceDialog .el-icon-circle-info:before{
	color:#CFCFCF;
}
#addIMEIConfigTask .el-pairgrid-title{
	top:10px;
}
#addIMEIConfigTask .el-form-item,.addDeviceDialog .el-form-item{
	display: flex;
	align-items: center;
}
#addIMEIConfigTask .el-form-item__content,.addDeviceDialog .el-form-item__content{
	margin-left: unset!important;
	width: 100%;
}
#addIMEIConfigTask .deviceTableBoxCls{
	display:flex;
	height:372px;
	width:100%;
}
#addIMEIConfigTask .deviceTableBoxCls .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
</style>
<!--enb新建IMEI配置任务  -->
<div id="addIMEIConfigTask">
	<el-form :model='ruleForm' :rules="rules" style="padding-top:20px;" ref="ruleForm" :hide-required-asterisk=true label-width="165px">
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item class="nameItem" label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' style='margin-left:45px;margin-top:18px;'>
			<el-input :disabled="showName" maxlength=100 v-model="ruleForm.taskname" size="mini" style="width:348px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<div style='margin-left:45px;width:93%;'>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' style="margin-bottom: 10px;">
				<el-radio-group v-model="deviceType" :disabled="showName" style="padding-top: 5px;">
					<el-radio border label="all"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio border label="select"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<div class="deviceTableBoxCls">
				<div style='border:1px solid #EFF0F2;flex:1;overflow:auto'>
					<el-pairgrid   
						:id="'select_device_list'" 
						:rownumber="true" 
						ref="add_task_table" 
						:right-url="rightUrl" 
						:left-url="leftUrl" 
						:height="height" 
						row-key="serialNumber" 
						:query-params="query_cell_params" 
						:title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" 
						@selection-change='selectChange' 
						@right-load-success="rightLoadSuccess"
						:readonly="showName"
					 >
						<template slot="left">
							<el-table-column type='selection' width="40"></el-table-column>
							<el-table-column prop="connectionStatus" width="50">
								<template slot-scope="scope">
									<div :class="{
										'el-icon el-icon-status-conn-off':scope.row.connectionStatus!='Exception' && scope.row.connectionStatus!='On' && scope.row.connectionStatus!='updating' && scope.row.connectionStatus!=1,
										'':scope.row.have_connected==2,
										'conn_exc':scope.row.connectionStatus=='Exception',
										'el-icon el-icon-status-conn-on':scope.row.connectionStatus=='On'||scope.row.connectionStatus=='updating'||scope.row.connectionStatus==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
								</template>
							</el-table-column>
							<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
							<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
							
						</template>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-query @query="query" type="normal" :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/IP'"></el-query>
						</template>
						<template slot='right'>
							<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
							<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
						</template>
					</el-pairgrid>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-bottom:20px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
			<div>
				<el-form-item prop='rawMode' style='margin-bottom:5px;' label="<%=rb.getString("WenJianLieBiao") %>" label-width="80px">
					
				</el-form-item>
			</div>
			<div style='border:1px solid #EFF0F2;'>
				<el-ctable :id="'select_file_list'" ref="add_file_table" :url="fileData" :default-checked="defaultChecked" @row-click="rowClickUpgrade" :url="fileUrl" :height="height" pagination="true" :query-params="fileParams" @load-success="loadSuccessFile">
					<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
						<template slot-scope="scope">
		              		<div class='tableDiv el-icon el-icon-status-yes selected-status' style="cursor: pointer;"></div>
		            	</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("WenJianMing")%>' width="500" prop="fileName"></el-table-column>
					<el-table-column label='<%=rb.getString("ShangChuanZhe")%>' width="200" prop="uploader"></el-table-column>
					<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' width="200" prop="uploadTime"></el-table-column>
					<el-table-column label='<%=rb.getString("MiaoShu")%>' prop="description"></el-table-column>
				</el-ctable>
			</div>
			<el-form-item prop='file'> 
				<el-input v-model='ruleForm.file' v-show="false"></el-input>
			</el-form-item>
		</div>
		<!--
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;margin-left:45px;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status" :disabled="showName">
				<el-radio border label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio border label="suspend"><%=rb.getString("GuaQi")%></el-radio>
                <el-radio border label="online"><%=rb.getString("ShangXianZhiXing")%></el-radio>
				<el-radio border label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;margin-bottom:32px;' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>-->
		<!-- 设备并发数 -->
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiBingFaShu")%></span>
		</div>
		<el-form-item style='display:inline-block;margin:20px 45px;width:50%;' label-width="220px" prop='maxConcurrentNumber' label="<%=rb.getString("SheBeiBingFaShu")%>">
			<el-input-number :disabled="showName" @change="maxConcurrentNumberChange" v-model='ruleForm.maxConcurrentNumber' :min="5" :max="100" style="width:200px;height:28px;line-height:28px;"></el-input-number>
		</el-form-item>
	</el-form>
	<el-dialog class='dialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='cancel'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='submit' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='cancel'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
var addUpgrade = new Vue({
	el:'#addIMEIConfigTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				callback();
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
		}
		var validateCodes = (rule,value,callback) => {
			if(this.deviceType == 'all'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		}
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
					return item.length > 0;
				});
	
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
			leftUrl:'',
			rightUrl:'',

			deviceTitle:['','<%=rb.getString("YiXuanZeJiZhan")%>'],
			height:'370px',
			width:'90%',
			fileUrl:'',
			fileParams:{
				timeZone:timeZone,
				searchText:''
			},
			setTimeEnable:true,
			rowData : [],
			ruleForm:{
				taskname:'${addTaskName}',
				cellCodes:'',
				file:'',
				status:'active',
				exetime:'',
				rawMode:true,
				maxConcurrentNumber:20,
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{validator:validateCodes,trigger:'change'}
				],
				file:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			defaultTaskName:'',
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			deviceUrl:'',
			query_cell_params:{
				searchText:'',
				connection_status: '',
				timeZone:timeZone
			},
			query_cell_form:{
				group_id:[],
				serial_number:'',
				host_name:'',
				software_version:[],
				module_type:'',
				rollback_version:[]
			},
			groupOptions:[],
			versionOptions:[],
			rbVersionOptions:[],
			firstFlag:true,
			firstFlagFile:true,
			showName:false,
			productOptions:[],
			deviceType:'select',
			showDeviceType:true,
			selection:[],
			showProduct:true,
			fileData:'${ctx}/cell/imei/queryImsiIMEIPageList.action',
			addListForm:{
				serialNumber:'',
				type:'input'
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:"<%=rb.getString("TianJia")%>",
			onlineOptions:[
				{label:"<%=rb.getString("QuanBu")%>",value:""},
				{label:"<%=rb.getString("ZaiXian")%>",value:"1"},
				{label:"<%=rb.getString("LiXian")%>",value:"0"}
			],
		}
	},
	methods:{ 
		init:function(){
			var vm = this;
			
			if(enbFileVue.operType == "viewTask"){
				vm.showName = true;
				axios.post('${ctx}/cell/imei/getImsiIMEITaskByTaskId.action',stringify({
					id : enbFileVue.rowDataTask.id,
				})).then(function(response){
					var data = response.data;
					vm.ruleForm.taskname = data.taskName;
					vm.ruleForm.file = data.fileId;
										
					
					vm.defaultTaskName = data.taskName;
					vm.deviceType = data.selDeviceType;
					vm.ruleForm.maxConcurrentNumber = data.deviceConcurrentNumber;

					vm.$nextTick(function(){
						
						vm.deviceUrl = '${ctx}/cell/imei/queryEnbInfoPageList.action';
						vm.leftUrl = '${ctx}/cell/imei/queryEnbInfoPageList.action';
						vm.$refs.ruleForm.clearValidate();
						vm.rightUrl = '${ctx}/cell/imei/queryEnbInfoPageList.action?taskId=' + enbFileVue.rowDataTask.id;
						vm.fileData ='${ctx}/cell/imei/queryImsiIMEIPageList.action';
						setTimeout(function(){
							vm.defaultChecked = [data.fileId+''];
							initForm(vm.$refs.ruleForm);
							
						},2000)
    		    	});

					
				})
			}else{
							
					
					vm.$nextTick(function(){
						if(!isJumpToPage) vm.$refs.add_task_table.appendCheckedRows(enbFileVue.cellData);
					});
								
				
			
				vm.$nextTick(function(){
					vm.deviceUrl = '${ctx}/cell/imei/queryEnbInfoPageList.action';
					vm.leftUrl = '${ctx}/cell/imei/queryEnbInfoPageList.action';
					
					vm.$refs.ruleForm.clearValidate();
					
					setTimeout(function(){
						initForm(vm.$refs.ruleForm);
					},2000)
				});
			}
			
			
		},
		query(val){
			this.query_cell_params.searchText = val;
		},
		
		rowClickUpgrade(row){
	        this.rowData = row;
	    },
		submit(){
	    	var vm = this;
	    	var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			if(vm.deviceType == 'all'){
	    				params.selDeviceType = "all"
	    			}else{
	    				params.selDeviceType = "select"
	    				params.serialNumbers = vm.ruleForm.cellCodes;
						
	    			}
	    			params.timeZone = timeZone;
	    			params.taskName = vm.ruleForm.taskname;
	    			params.status = vm.ruleForm.status;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
					
	    			params.fileId = vm.ruleForm.file;
	    		 			
	    		
	    			params.deviceConcurrentNumber = vm.ruleForm.maxConcurrentNumber;
	    			if(enbFileVue.operType == "modifyTask"){
	    				url = '${ctx}/task/upgrade/updateTask.action'
	    	    		params.taskId = enbFileVue.rowDataTask.TASK_ID
    	    			if(!isFormChanged(vm.$refs.ruleForm)){
    						vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
    							confirmButtonText:'<%=rb.getString("QueDing")%>',
    							type:'warning'
    						}).then().catch();
    						return;
       	    			}
	    	    		vm.saveTask(url,params,message);
	    			}else{
	    				url = '${ctx}/cell/imei/addImsiIMEITask.action'
       	    			vm.$confirm("<%=rb.getString("QueRenXinJianRenWu")%>",'<%=rb.getString("QueRen")%>',{
       						customClass:'warningConfirm',
       						confirmButtonText:'<%=rb.getString("QueDing")%>',
       						cancelButtonText:'<%=rb.getString("QuXiao")%>',
       						type:'warning',
       						closeOnClickModal:false
       					}).then(() => {
       						vm.saveTask(url,params,message);
       					}).catch(() => {})
	    			}
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(url,params,message){
			var vm = this;
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    enbFileVue.$refs.slide.hide();
                    enbFileVue.$refs.upgrade_cell_table.clearSelection();
                    enbFileVue.$refs.upgrade_task_table.refresh();
                    isJumpToPage = '';
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(this.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					enbFileVue.$refs.slide.hide();
					isJumpToPage = '';
				}).catch(() => {})
			}else{
				enbFileVue.$refs.slide.hide();
				isJumpToPage = '';
			}
		},
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		loadSuccessFile(data){
			var vm = this;
			if(enbFileVue.operType == "modifyTask" || enbFileVue.operType == "viewTask"){
				axios.post('${ctx}/task/upgrade/getTask.action',stringify({
					taskId : enbFileVue.rowDataTask.TASK_ID,
					timeZone : timeZone
				})).then(function(response){
					var result = response.data;
					data.rows.map(function(item){
						if(result.FILE_ID == item.id){
							vm.$refs.add_file_table.setCurrentRow(item)
						}
					})
					if(vm.firstFlagFile){
						initForm(vm.$refs.ruleForm);
						vm.firstFlagFile = false;
					}
				})
			}
			if(isJumpToPage){
				data.rows.map(function(item){
					if(isJumpToPage.vid == item.id){
						vm.$refs.add_file_table.setCurrentRow(item);
						vm.ruleForm.file = isJumpToPage.vid;
					}
				})
			}
		},
		selectChange(selection){
			this.selection = selection;
		},
		rightLoadSuccess(){
			var vm = this;
			if(vm.firstFlag){
				initForm(vm.$refs.ruleForm);
				vm.firstFlag = false;
			}
		},
		connectionStatusChange(val){
			var vm = this;
			
				vm.query_cell_params.connection_status = val;
			
		},
		
		// 最大并发数失焦事件
		maxConcurrentNumberChange(event){
			var vm = this,
				value = vm.ruleForm.maxConcurrentNumber;
			if(value == undefined || value == '' || value == null){
				vm.$nextTick(()=>{
					vm.ruleForm.maxConcurrentNumber = 5;
				})
			}
		},
	},
	watch:{
		rowData(newVal){
			this.ruleForm.file = newVal.id
		},
		"ruleForm.status":function(newVal){
			if(enbFileVue.operType =='viewTask' || newVal != 'timing'){
				this.setTimeEnable = true;
			}else{
				this.setTimeEnable = false;
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		
		deviceType:function(val){
			if(val == 'all'){
				this.showDeviceType = false;
			}else{
				this.showDeviceType = true;
			}
			this.resetQuery();
		},
		
		selection(){
			var vm = this,
			cellCodeList=[],
			euSerialNumberList=[],
			ruSerialNumberList=[]; 

			
				var data = vm.$refs.add_task_table.getData();
				if(data.length != 0){
					data.map(function(item){
						cellCodeList.push(item.serialNumber);
					})
				}
				vm.ruleForm.cellCodes = cellCodeList.join(',');
			
		}
	},
	mounted(){
		this.init();
		eventBus.$off('add-task').$on('add-task',this.submit);
		eventBus.$off('cancel-add-task').$on('cancel-add-task',this.cancel);
	}
})
</script>