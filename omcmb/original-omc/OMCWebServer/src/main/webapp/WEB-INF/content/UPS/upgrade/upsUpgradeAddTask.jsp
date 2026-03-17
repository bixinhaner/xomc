<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#upsUpgradeAddTask .el-form-item__label{
	line-height:26px;
}
#upsUpgradeAddTask .modeItem .el-radio{
	display:block;
	margin-left:0px;
	margin-bottom:20px;
} 
#upsUpgradeAddTask .modeItem{
	margin-top:20px;
}
#upsUpgradeAddTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
.deviceItem{
	display:inline-block;
	margin-right:60px;
}
.alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
.titleStyML{
	margin-left: 20px;
}
</style>

<%-- 新建UPS升级任务 --%>
<div id="upsUpgradeAddTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:20px;' label-position="left"  :hide-required-asterisk=true>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' label-width="120px" prop='taskname' style='margin-left:45px;margin-top:20px;'>
			<el-input maxlength=100 :disabled='viewFlag' v-model="ruleForm.taskname" size="mini" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-pairgrid style='margin:10px 45px 0px 45px;' :id="'select_device_list'" v-if='showPairGrid' :rownumber="true" ref="cpairgrid" @selection-change='selectChange'  :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="serial_number" :query-params="queryParams" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
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
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
				<el-table-column prop="device_name" label='<%=rb.getString("Title_SheBeiMingCheng") %>' min-width="110" show-overflow-tooltip ></el-table-column>
				<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' min-width="200"></el-table-column>
				<el-table-column prop='group_id' :formatter="deviceGroupFormatter" label='<%=rb.getString("SheBeiZu")%>' min-width="200"></el-table-column>
			</template>
			<template slot='toolbar'>
				<el-form :model='queryForm' ref="queryForm" label-position="top">
					<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>'"
					:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
						<template slot="form">
							<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serialNumber'>
								<el-input v-model='queryForm.serialNumber'  size="mini"></el-input>
							</el-form-item>
							<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
								<el-select v-model="queryForm.group_id" size="mini">
									<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
									</el-option>
								</el-select>
							</el-form-item>
							<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='swVersion'>
								<el-select v-model="queryForm.swVersion" size="mini">
									<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
									</el-option>
								</el-select>
							</el-form-item>
						</template>
					</el-query>
				</el-form>
			</template>
			<template slot='right'>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
			</template>
		</el-pairgrid>
		<div v-else style='margin:10px 45px 0px 45px;'>
			<label style='font-size:16px;'><%=rb.getString("YiXuanSheBei")%></label>
			<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;" ref="stable"  :url="rightUrl" :height="height" pagination="true" :query-params="fileParams">
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
				<el-table-column prop="device_name" label='<%=rb.getString("Title_SheBeiMingCheng") %>' min-width="110" show-overflow-tooltip ></el-table-column>
				<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' min-width="200"></el-table-column>
				<el-table-column prop='group_name'  label='<%=rb.getString("SheBeiZu")%>' min-width="200"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='cellCodes' style='margin-left:45px;'>
			<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("WenJianLieBiao")%></span>
		</div>
		<div class='fileContent' style='margin:10px 45px 0px 45px;border:1px solid #E9E9E9;'>
			<el-ctable :id="'select_file_list'" row-key="id" :readonly='viewFlag' ref="ctable" :url='fileDateUrl' :default-checked="defaultChecked" @row-click="rowClickUpgrade" @load-success='loadSuccess' :height="height" pagination="true" :query-params="fileParams">
				<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
					<template slot-scope="scope">
	              		<div class='tableDiv el-icon el-icon-status-yes selected-status' style="cursor: pointer;"></div>
	            	</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("BanBen")%>' width="200"  prop="version"></el-table-column>
				<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="200" prop="product"></el-table-column>
				<el-table-column label='<%=rb.getString("WenJianMing")%>' prop="file_name"></el-table-column>
				<el-table-column label='<%=rb.getString("WenJianDaXiao")%>' width="150" prop="size"></el-table-column>
				<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' width="200" prop="upload_time"></el-table-column>
				<el-table-column label='<%=rb.getString("MiaoShu")%>' prop="desc"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='fileIds' style='margin-left:45px;'>
			<el-input v-model='ruleForm.fileIds' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item  prop='status'>
				<el-radio-group v-model="ruleForm.status" style='margin-top:17px;' :disabled='viewFlag'>
					<el-radio label="active" style='margin-right:110px;margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio label="suspend" style='margin-right:110px;'><%=rb.getString("GuaQi")%></el-radio>
					<el-radio label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;' >
				<el-date-picker style='margin-top:10px;vertical-align:middle;margin-left:25px;' value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="viewFlag||setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
</div>

<script type="text/javascript">
new Vue({
	el:'#upsUpgradeAddTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/upgrade/ups/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskname.trim(),
					taskType:'1'
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
				if(value == '' || value==null){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		}
		return {
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			groupOptions:[],
			versionOptions:[],
			fileParams:{
				timeZone:timeZone,
				isShowSlave:false,
				file_type:'5',
				productValue:'',
				fileId:'',
			},
			deviceTitle:['<%=rb.getString("UPSLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskname:'${addTaskName}',
				cellCodes:'',
				fileIds:'',
				fileName:'',
				status:'active',
				exetime:'',
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
				],
				fileIds:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			queryParams:{
				group_id:'',
				serialNumber:'',
				searchText:'',
				swVersion:''
			},
			queryForm:{
				group_id:'',
				serialNumber:'',
				swVersion:''
			},
			taskId:'',
			operationType:'',
			defaultTaskName:'',
			showPairGrid:true,
			fileDateUrl:'',
			viewFlag:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.getNow()-8.64e7;
				}
			},
		}
	},
	methods:{ 
		// 初始化
		init(id,type){
			var vm = this;
			vm.taskId = id ;
			vm.operationType = type;
			if(type == 'taskView'){
				vm.viewFlag = true;
				vm.showPairGrid = false;
			}
			
			axios.post('${ctx}/ups/querySWVersionList.action').then(function(response){
				let data = response.data
				vm.versionOptions = data;
				
			}).catch(function(error){})
			axios.post('${ctx}/ups/queryDeviceGroupList.action').then(function(response){
				let data = response.data
				vm.groupOptions = data;
				vm.leftUrl='${ctx}/ups/queryUpsDevices.action';
				if(type !== "addTask"){
					vm.rightUrl='${ctx}/task/upgrade/ups/getTaskSelectedList.action?taskId='+vm.taskId;
					vm.getTaskDateInfo();
				}else{
					vm.fileDateUrl = '${ctx}/cell/version/queryfileInfosList.action';
				}
				initForm(vm.$refs.ruleForm);
			}).catch(function(error){})
		},
		// 模糊查询
		query(val){
			var vm = this;
			vm.resetQuery();
			vm.queryParams.searchText = val;
			vm.$refs.cpairgrid.reset();
		},
		// 高级查询
		advanceQuery(){
			var vm = this;
			vm.queryParams.searchText = '';
			vm.queryParams.serialNumber = vm.queryForm.serialNumber;
			vm.queryParams.group_id = vm.queryForm.group_id;
			vm.queryParams.swVersion = vm.queryForm.swVersion;
			vm.$refs.cpairgrid.reset();
		},
		// 高级查询重置
		resetQuery(){
			var vm = this;
			vm.queryForm.serialNumber = '';
			vm.queryForm.group_id = '';
			vm.queryForm.swVersion = '';
		},
		// UPS设备选择事件
		selectChange(selection){
			this.selection = selection
		},
		// 升级文件选择事件
		rowClickUpgrade(row){
	        this.rowData = row;
	    },
		// 设备组名称格式化
		deviceGroupFormatter(row,column,cellValue,index){
			var vm = this,groupName = '';
			vm.groupOptions.map((item)=>{
				if(item.value == cellValue){
					groupName = item.text
				}
			})
			return groupName
		},
		// 获取UPS升级任务详情
		getTaskDateInfo(){
			var vm = this,
				params={
					taskId:vm.taskId,
					timeZone:timeZone
				};
			
			axios.post('${ctx}/task/upgrade/ups/getTask.action',stringify(params)).then(function(response){
				let data = response.data;
				vm.ruleForm.taskname = data.TASK_NAME;
				vm.ruleForm.fileIds = data.FILE_ID;
				vm.ruleForm.status = data.TASK_STATUS;
				if(data.TASK_STATUS == 'timing'){
					vm.ruleForm.exetime = data.START_TIME;
				}
				vm.defaultChecked = [data.FILE_ID+''];
				vm.defaultTaskName = data.TASK_NAME;
				if(vm.operationType == 'taskView'){
					vm.fileParams.fileId = data.FILE_ID;
				}
				vm.fileDateUrl = '${ctx}/cell/version/queryfileInfosList.action';
				
			}).catch(function(error){})
		},
		// 提交
		submit(){
	    	var vm = this;
	    	var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},urls='';
	    			params.timeZone = timeZone;
	    			params.upsCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskname;
					params.productValue = '1';
					params.taskType = '1';
					params.rawMode = 'false';
	    			params.status = vm.ruleForm.status;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.file_id = vm.ruleForm.fileIds;
					params.file_name = vm.ruleForm.fileName;
					if(vm.operationType == 'addTask'){
						urls = '${ctx}/task/upgrade/ups/addTask.action';
					}else{
						params.taskId = vm.taskId
						urls = '${ctx}/task/upgrade/ups/updateTask.action';
					}
					axios.post(urls,stringify(params)).then(function(response){
						var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            eventBus.$emit('hide-upsUpgrade-slide')
	    				}else{
	    					vm.$message.error(data["message"])
	    				}
					}).catch(function(error){})
	    			
	    		}else{
	    			return false;
	    		}
	    	})
		},
		// 取消
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
					eventBus.$emit('hide-upsUpgrade-slide')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-upsUpgrade-slide')
			}
		},
		// ups设备表格加载成功回调
		loadSuccess(data){
			var vm = this;
			if(vm.operationType !== 'addTask'){
				var tb = vm.$refs.ctable;
				vm.updateRow(data.rows,tb);
			}
		},
		updateRow(rows,tb){
			var vm = this;
			axios.post('${ctx}/task/upgrade/ups/getTask.action',stringify({
				taskId : vm.taskId,
				timeZone : timeZone
			})).then(function(response){
				var data = response.data;
				rows.map(function(item){
					if(item.id == data.FILE_ID){
						tb.setCurrentRow(item)
					}
				})
			})
		},
		// 定时时间失焦事件
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		
	},
	watch:{
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var cellCodes = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.ups_code + ","
				})
			}
			this.ruleForm.cellCodes = cellCodes
		},
		rowData(newVal){
			this.ruleForm.fileIds = newVal.id;
			this.ruleForm.fileName = newVal.file_name;
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		}
	},
	mounted(){
		eventBus.$off('ups-upgrade-taskInit').$on('ups-upgrade-taskInit',this.init);
		eventBus.$off('ups-upgrade-addSubmit').$on('ups-upgrade-addSubmit',this.submit);
		eventBus.$off('ups-upgrade-addCancel').$on('ups-upgrade-addCancel',this.cancel);
	}
})
</script>