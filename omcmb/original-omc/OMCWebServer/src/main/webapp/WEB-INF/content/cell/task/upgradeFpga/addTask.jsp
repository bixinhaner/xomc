<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#addFpgaTask .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#addFpgaTask .modeItem .el-radio{
	display:block;
	margin-left:0px;
	margin-bottom:20px;
}
#addFpgaTask .modeItem{
	margin-top:20px;
}
#addFpgaTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#addFpgaTask .el-form-item{
	margin-left:55px;
}
</style>

<%-- 新建升级任务 --%>
<div id="addFpgaTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:35px;' :hide-required-asterisk=true>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname'>
			<el-input maxlength=100 :disabled='showName' v-model="ruleForm.taskname" size="mini" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:380px;'>
			<el-select v-model='ruleForm.product' :disabled='showProduct'>	
				<el-option v-for='item in productType' :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-pairgrid :id="'select_device_list'" v-if='showPairGrid' ref="cpairgrid" @selection-change='selectChange' :query-name="queryName"  :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="serial_number" :query-params="queryForm" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" style='margin-left:55px;'>
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
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width="300"></el-table-column>
				<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' width="200"></el-table-column>
				<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHao")%>' width="150"></el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' width="200"></el-table-column>
			</template>
			<template slot='toolbar'>
				<el-form :model='queryForm' ref="queryForm" label-position="top">
					<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
					:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
						<template slot="form">
							<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
								<el-input v-model='queryForm.serial_number'  size="mini" style='width:200px'></el-input>
							</el-form-item>
							<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
								<el-input v-model='queryForm.host_name' size="mini" style='width:200px'></el-input>
							</el-form-item>
							<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
								<el-select v-model="queryForm.group_id">
									<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
									</el-option>
								</el-select>
							</el-form-item>
							<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
								<el-select v-model="queryForm.software_version">
									<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
									</el-option>
								</el-select>
							</el-form-item>
							<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
								<el-input v-model='queryForm.module_type'  size="mini"></el-input>
							</el-form-item>
						</template>
					</el-query>
				</el-form>
			</template>
			<template slot='right'>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
		</el-pairgrid>
		<div v-else style='margin-left:55px;'>
			<label style='font-size:16px;'><%=rb.getString("YiXuanZeJiZhan")%></label>
			<el-ctable :id="'selected_device_list'" ref="stable"  :url="rightUrl" :height="height" pagination="true" :query-params="fileParams">
					<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
					<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='cellCodes'>
			<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("WenJianLieBiao")%></span>
		</div>
		<div class='fileContent' style='margin-left:55px;'>
			<el-ctable ref="ctable" :data='fileData' :default-checked="defaultChecked" @row-click="rowClickUpgrade" @load-success='loadSuccess' :url="fileUrl" :height="height" pagination="true" :query-params="fileParams">
					<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
						<template slot-scope="scope">
	              			<div class='tableDiv el-icon el-icon-status-yes selected-status'   style="cursor: pointer;"></div>
	            		</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("BanBen")%>' width="200"  prop="version"></el-table-column>
					<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="200" prop="product"></el-table-column>
					<el-table-column label='<%=rb.getString("WenJianMing")%>' prop="file_name"></el-table-column>
					<el-table-column label='<%=rb.getString("WenJianDaXiao")%>' width=150" prop="size"></el-table-column>
					<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' width="200" prop="upload_time"></el-table-column>
					<el-table-column label='<%=rb.getString("MiaoShu")%>' width="200" prop="desc"></el-table-column>
			</el-ctable>
			<el-form-item prop='file'>
				<el-input v-model='ruleForm.file' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status" :disabled='showType'>
				<el-radio label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio label="suspend"><%=rb.getString("GuaQi")%></el-radio>
				<el-radio label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus="setTime" :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
	</el-form>
</div>

<script type="text/javascript">
new Vue({
	el:'#addFpgaTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/upgrade/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskname.trim(),
					taskType:'6'
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
				if(value == '' || value == null){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		}
		return {
			productType: [],
			leftUrl:'',
			rightUrl:'',
			queryName:'serial_number,host_name',
			height:'370px',
			groupOptions:[],
			versionOptions:[],
			fileUrl:'',
			fileParams:{
				timeZone:timeZone,
				productValue:'',
				isShowSlave:false
			},
			deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskname:'${addTaskName}',
				product:'',
				cellCodes:'',
				file:'',
				status:'active',
				exetime:'',
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				product:[
					{required:true,message:'<%=rb.getString("QingXuanZeChanPinLeiXing")%>',trigger:'change'}
				],
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
				],
				file:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			defaultProduct:'',
			queryForm:{
				group_id:'',
				serial_nubmer:'',
				host_name:'',
				software_version:'',
				productValue:'',
				search_text:'',
				module_type:''
			},
			taskId:'',
			defaultTaskName:'',
			task_type:'add',
			showPairGrid:true,
			fileData:'',
			showName:false,
			showProduct:false,
			showType:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			}
		}
	},
	methods:{ 
		init:function(){
			var vm = this;
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.ruleForm.product,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.ruleForm.product,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){
				
			})
		},
		query(val){
			this.resetQuery();
			this.queryForm.search_text = val;
			this.$refs.cpairgrid.reload();
		},
		advanceQuery(){
			this.queryForm.search_text = "";
			this.$refs.cpairgrid.reload();
		},
		resetQuery(){
			this.$refs.queryForm.resetFields();
		},
		selectChange(selection){
			this.selection = selection
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
	    			params.timeZone = timeZone;
	    			params.cellCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskname;
	    			params.status = vm.ruleForm.status;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.fileId = vm.ruleForm.file;
	    			params.taskType = "6";
	    			params.productValue = vm.ruleForm.product;
	    			if(vm.task_type == 'edit'){
	    	    		url = '${ctx}/task/upgrade/updateTask.action'
	    	    		params.taskId = vm.taskId
    	    			if(!isFormChanged(vm.$refs.ruleForm)){
    						vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
    							confirmButtonText:'<%=rb.getString("QueDing")%>',
    							type:'warning'
    						}).then().catch();
    						return;
       	    			}
	    	    		vm.saveTask(url,params,message);
	    	    	}else if(vm.task_type == 'add'){
	    	    		url = '${ctx}/task/upgrade/addTask.action'
    	    			vm.$confirm("<%=rb.getString("QueRenXinJianRenWu")%>",'<%=rb.getString("QueRen")%>',{
    						customClass:'warningConfirm',
    						confirmButtonText:'<%=rb.getString("QueDing")%>',
    						cancelButtonText:'<%=rb.getString("QuXiao")%>',
    						type:'warning',
    						closeOnClickModal:false
    					}).then(() => {
    						vm.saveTask(url,params,message);
    					}).catch(() => {
    						
    					})
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
                    if(vm.task_type == 'add'){
                        eventBus.$emit('to-task-view');//调整到任务查看页面
                    }else if(vm.task_type == 'edit'){
                        eventBus.$emit('cancel-upgrade');
                    }
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
					if(vm.task_type == 'add'){
						eventBus.$emit('hide-upgrade')
					}else{
						enbSoftUpgradeVM.$refs.slide.hide();
					}
				}).catch(() => {
					
				})
			}else{
				if(vm.task_type == 'add'){
					eventBus.$emit('hide-upgrade')
				}else{
					enbSoftUpgradeVM.$refs.slide.hide();
				}
			}
		},
		loadSuccess(data){
			if(this.task_type == 'edit'){
				var tb = this.$refs.ctable;
				this.updateRow(data.rows,tb);
			}
		},
		updateRow(rows,tb){
			var vm = this;
			axios.post('${ctx}/task/upgrade/getTask.action',stringify({
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
		modifyTask(task_id,taskStatus){
			var vm = this;
			vm.taskId = task_id;
			axios.post('${ctx}/cell/version/getProductType.action').then(function(response){
				vm.productType = response.data;
				axios.post('${ctx}/task/upgrade/getTask.action',stringify({
					taskId : task_id,
					timeZone : timeZone
				})).then(function(response){
					var data = response.data;
					vm.ruleForm.taskname = data.TASK_NAME;
					//var reg = new RegExp('w',"g");
					vm.ruleForm.product = data.PRODUCT_TYPE;
					vm.$nextTick(function(){
						vm.rightUrl = '${ctx}/task/upgrade/getTaskSelectedList.action?taskId=' + task_id;
    		    	});
					vm.ruleForm.file = data.FILE_ID;
					vm.ruleForm.status = data.CREATE_STATUS;
					if(data.CREATE_STATUS == 'timing'){
						vm.ruleForm.exetime = data.CREATE_TIME;
					}
					vm.defaultProduct = data.PRODUCT_TYPE;
					vm.defaultChecked = [data.FILE_ID+''];
					vm.defaultTaskName = data.TASK_NAME;
					vm.task_type = 'edit';
					setTimeout(function(){
    					initForm(vm.$refs.ruleForm);
    				},500);
					if(taskStatus != 1){
						vm.showPairGrid = false;
						vm.showName = true;
						vm.showProduct = true;
						vm.showType = true;
						setTimeout(function(){
							vm.setTimeEnable = true;
						},20)
					}
				})
			}).catch(function(error){
				
			})
		},
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime')
		},
		addTask(){
			var vm = this;
			axios.post('${ctx}/cell/version/getProductType.action').then(function(response){
				vm.productType = response.data;
				vm.ruleForm.product = vm.productType[0].value;
				initForm(vm.$refs.ruleForm);
			}).catch(function(error){
				
			})
		}
	},
	watch:{
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var cellCodes = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.small_cell_code + ","
				})
			}
			this.ruleForm.cellCodes = cellCodes
		},
		"ruleForm.product":function(newVal){
			var vm = this;
			if(newVal != vm.defaultProduct){
				vm.ruleForm.file = ''
			}else{
				vm.ruleForm.file = vm.defaultChecked;
			}
			var reg = new RegExp('\\\\',"g");
			if(vm.showPairGrid){
				vm.queryForm.productValue = newVal;
				vm.$refs.cpairgrid.clear();
				vm.queryForm.productValue = newVal.replace(reg,'');
				vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action'
				vm.$refs.cpairgrid.reload();
			}
			vm.fileParams.productValue = newVal.replace(reg,'');
			if(vm.showPairGrid){
				vm.fileUrl = '${ctx}/cell/version/queryfileInfosList.action?file_type=6'
				vm.$refs.ctable.refresh();
			}else{
				vm.fileUrl = ''
				axios.post('${ctx}/cell/version/queryfileInfosList.action?file_type=6',stringify({
					timeZone:timeZone,
					productValue:newVal.replace(reg,''),
					isShowSlave:false,
					fileId:vm.ruleForm.file,
					page:1,
					rows:10,
					sort:"",
					order:""
				})).then(function(response){
					var data = response.data;
					vm.fileData = data.rows;
					setTimeout(function(){
						vm.$refs.ctable.setCurrentRow(vm.fileData[0])
					},500)
				})
			}
			if(newVal == vm.defaultProduct){
				vm.$refs.cpairgrid.reloadRightTb();
			}
			var reg = new RegExp('\\\\',"g");
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : newVal.replace(reg,''),
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : newVal.replace(reg,''),
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){
				
			})
		},
		rowData(newVal){
			this.ruleForm.file = newVal.id
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
		this.init();
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('hander-cancel').$on('hander-cancel',this.cancel);
		eventBus.$off('modify-task').$on('modify-task',this.modifyTask);
		eventBus.$off('add-task').$on('add-task',this.addTask);
	}
	
})
</script>