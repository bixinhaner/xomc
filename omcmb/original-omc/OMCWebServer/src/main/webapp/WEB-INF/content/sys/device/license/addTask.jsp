<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.placeholder-bt[placeholder]:hover::after {
		width: 75px;
	}
	.queryInfo{
		display:inline-block;
		margin-right:40px;
	}
	.queryInfo label{
		display:block;
		margin-bottom:5px;
		line-height:26px;
	}
	.tableDiv{
		height:23px;
	}
    #addLicenseTask .executeModeBoxCls{
        height:60px;
        width:100%;
        border:1px solid #D1ECF5;
        display:flex;
        align-items: center;
    }
</style>
<div id='addLicenseTask' style='padding: 30px 40px;'>
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" label-position="top">
		<div class="group-title not-extend" style='padding-bottom: 20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' style='margin-left:30px;' prop='task_name'>
			<el-input v-model='ruleForm.task_name' :disabled='taskNameDisabled' size='mini' style="width:800px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div class="group-title not-extend" style='padding-bottom: 20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("WenJianLieBiao")%></span>
		</div>
		<div style='height:300px;margin-left:30px;margin-bottom:10px;'>
			<el-ctable :id="'licenseFileTable'" :url='fileUrl' :default-checked="defaultChecked" @row-click="clickFile" ref="ctable" height='100%' pagination="true">
				<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
					<template slot-scope="scope">
	              		<div class='tableDiv el-icon-check selected-status' style="cursor: pointer;"></div>
	            	</template>
				</el-table-column>
				<el-table-column prop='file_name' label='<%=rb.getString("WenJianMing")%>' show-overflow-tooltip="true"></el-table-column>
				<el-table-column prop='mac_range' label='<%=rb.getString("MACDiZhi")%>' show-overflow-tooltip="true"></el-table-column>
				<el-table-column prop='description' label='<%=rb.getString("MiaoShu")%>'></el-table-column>
			</el-ctable>
			<el-form-item prop='file'>
				<el-input v-show=false v-model='ruleForm.file'></el-input>
			</el-form-item>
		</div>
		<div class="group-title not-extend" style='padding-bottom: 20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<div style='margin-left:30px;'>
			<el-pairgrid v-if='viewTaskFlag' :id="'licenseSelected'" :rownumber="true" ref="cpairgrid" @selection-change='selectChange' height='340px' :right-url="rightUrl" :left-url="leftUrl" row-key="small_cell_code" :query-params="queryParams" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
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
					<el-table-column prop='small_cell_code' v-if=false></el-table-column>
					<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
					<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width="100"></el-table-column>
					<el-table-column prop='mac_address' label='<%=rb.getString("MACDiZhi")%>'></el-table-column>
				</template>
				<template slot='toolbar'>
					<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
					:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
						<template slot="form">
							<div class='queryInfo'>
								<label><%=rb.getString("XiaoZhanBianMa")%></label>
								<el-input v-model='queryParams.serial_number' size='mini'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("HostName")%></label>
								<el-input v-model='queryParams.host_name' size='mini'></el-input>
							</div>
							<div class='queryInfo'>
								<label><%=rb.getString("MACDiZhi")%></label>
								<el-input v-model='queryParams.mac_address' size='mini'></el-input>
							</div>
						</template>
					</el-query>
				</template>
				<template slot='right'>
					<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width='200'></el-table-column>
					<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width='100'></el-table-column>
					<el-table-column prop='mac_address' label='<%=rb.getString("MACDiZhi")%>' width='200'></el-table-column>
				</template>
			</el-pairgrid>
			<el-ctable v-else :url='rightUrl'  ref="viewCtable" height='340px' pagination="true">
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' show-overflow-tooltip="true"></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' show-overflow-tooltip="true"></el-table-column>
				<el-table-column prop='mac_address' label='<%=rb.getString("MACDiZhi")%>'></el-table-column>
			</el-ctable>
			<el-form-item prop='cellCodes'>
				<el-input v-show=false v-model='ruleForm.cellCodes'></el-input>
			</el-form-item>
		</div>
		<div class="group-title not-extend" style='padding-bottom: 20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div class="executeModeBoxCls">
			<el-form-item prop='status' style="margin-bottom: 0px;">
				<el-radio-group v-model='ruleForm.status' :disabled='statusDisabled'>
					<el-radio label='active' style='margin-right:50px;margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio label='suspend' style='margin-right:50px;'><%=rb.getString("GuaQi")%></el-radio>
					<el-radio label='timing'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' class='errorMsg' style="margin-bottom: 0px;">
				<el-date-picker v-model='ruleForm.exetime' style='vertical-align:middle;margin-left:25px;' :disabled="setTimeEnable" value-format="yyyy-MM-dd HH:mm:ss" type="datetime"  @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
</div>
<script>
	var addLic = new Vue({
		el:'#addLicenseTask',
		data(){
			var vm = this;
			var validateName = (rule,value,callback) => {
				if(value === ''){
					callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
				}else if(value.trim() == vm.defaultTaskName){
					callback();
				}else{
					axios.post('${ctx}/cell/1588License/taskNameExist.action',stringify({
						task_name:vm.ruleForm.task_name.trim()
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
			return{
				ruleForm:{
					task_name:'${addTaskName}',
					file:'',
					cellCodes:'',
					cellName:'',
					status:'active',
					exetime:''
				},
				rules:{
					task_name:[
						{validator:validateName,trigger:'blur'}
					],
					file:[
						{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
					],
					cellCodes:[
						{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
					],
					exetime:[
						{type:'date',validator:validateTime,trigger:'change'}
					]
				},
				leftUrl:'',
				rightUrl:'',
				queryParams:{
					search_text:'',
					host_name:'',
					serial_number:'',
					mac_address:'',
					mac_range:''
				},
				deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
				setTimeEnable:true,
				pickerOptions:{
					disabledDate(time){
						return time.getTime()< Date.now()-8.64e7;
					}
				},
				fileUrl:'${ctx}/cell/1588License/getFileInfoList.action',
				defaultTaskName:'',
				fileData:[],
				oper_type:'add',
				defaultChecked:[],
				task_id:'',
				viewTaskFlag:true,
				viewFileData:[],
				statusDisabled:false,
				taskNameDisabled:false
			}
		},
		methods:{
			selectChange(selection){
				var cellCodes = '';
				var cellName = '';
				if(selection.length != 0){
					selection.map(function(item){
						cellCodes += item.small_cell_code + ",";
						cellName += item.host_name + ','
					})
				}
				cellCodes = cellCodes.substring(0,cellCodes.length-1);
				cellName = cellName.substring(0,cellName.length-1);
				this.ruleForm.cellCodes = cellCodes;
				this.ruleForm.cellName = cellName;
			},
			query(val){
				this.resetQuery();
    			this.queryParams.search_text  = val;
    			this.$refs.cpairgrid.reload();
			},
			advanceQuery(){
				this.queryParams.search_text = "";
    			this.$refs.cpairgrid.reload()
			},
			resetQuery(){
				this.queryParams.serial_number = '';
				this.queryParams.host_name = '';
				this.queryParams.mac_address = '';
			},
			setTime(){
				this.ruleForm.exetime = formatDate(new Date(gloableTime));
				this.$refs.ruleForm.validateField('exetime');
			},
			clickFile(row){
				this.fileData = row;
			},
			submit(){
				var vm = this;
		    	var message = '';
		    	vm.$refs.ruleForm.validate((valid) => {
		    		if(valid){
		    			var data = vm.$refs.cpairgrid.getData();
		    			var serialNumber = '';
		    			var macStr = '';
						if(data.length != 0){
							data.map(function(item){
								serialNumber += item.serial_number + ",";
								macStr += item.mac_address + ",";
							})
						}
						serialNumber = serialNumber.substring(0,serialNumber.length-1);
						macStr = macStr.substring(0,macStr.length-1);
		    			var params = {};
		    			params.time_zone = timeZone;
		    			params.smallCellStr = vm.ruleForm.cellCodes;
		    			params.hostNameStr = vm.ruleForm.cellName;
		    			params.serialNumberStr = serialNumber;
		    			params.macStr = macStr;
		    			params.task_name = vm.ruleForm.task_name;
		    			params.task_status = vm.ruleForm.status;
		    			if(vm.ruleForm.status == 'timing'){
		    				params.time = vm.ruleForm.exetime;
		    			}
		    			params.file_id = vm.ruleForm.file;
		    			params.file_name = vm.fileData.file_name;
		    			if(vm.oper_type == 'edit'){
		    	    		url = '${ctx}/cell/1588License/updateTask.action'
		    	    		params.task_id = vm.task_id;
		    	    		message = '<%=rb.getString("XiuGaiRenWuChengGong")%>'
		    	    	}else if(vm.oper_type == 'add'){
		    	    		url = '${ctx}/cell/1588License/addTask.action'
		    	    		message = '<%=rb.getString("XinJianRenWuChengGong")%>'
		    	    	}

						// 检测设备是否已在任务中
						axios.post('${ctx}/cell/1588License/taskCellExist.action',stringify({
							smallCellStr: params.smallCellStr,
							task_id: params.task_id||''
						})).then(function(res){
							var data = res.data;
							
							if(data && data.rows.length) {
								var isUsed = false
									tNames = [];
								
								data.rows.map(function(row){
									if(row.progress_status != 'End'){
										isUsed = true;
										if(!tNames.includes(row.task_name)) tNames.push(row.task_name);
									}
								});
								if(isUsed){
									var msgInfo = '<%=rb.getString("RenWuSheBeiJianCePrev")%>'+tNames.join('、')+'<%=rb.getString("RenWuSheBeiJianCeSuffix")%>';
									toast(msgInfo,$('#omc_app_ctn'));
								}else{
									vm.$confirm('<%=rb.getString("SheBeiChongFuXuanZhe")%>','<%=rb.getString("QueRen")%>',{
										type: 'warning'
									}).then(function(){
										axios.post(url,stringify(params)).then(function(response){
											var data = response.data;
											if(data["success"]){
												vm.$message({
													message:message,
													type:'success',
												})
                                                if(vm.oper_type == 'add'){
                                                    eventBus.$emit('save-suc');
                                                    eventBus.$emit('to-task-view');//调整到任务查看页面
                                                }else if(vm.oper_type == 'edit'){
                                                    eventBus.$emit('save-edit-suc');
                                                }
											}else{
												vm.$message.error(data["message"])
											}
										})
									}).catch(function(){});
								}
							}else{
								axios.post(url,stringify(params)).then(function(response){
									var data = response.data;
									if(data["success"]){
										vm.$message({
											message:message,
											type:'success',
										})
                                        if(vm.oper_type == 'add'){
                                            eventBus.$emit('save-suc');
                                            eventBus.$emit('to-task-view');//调整到任务查看页面
                                        }else if(vm.oper_type == 'edit'){
                                            eventBus.$emit('save-edit-suc');
                                        }
									}else{
										vm.$message.error(data["message"])
									}
								})
							}
						});
		    		}else{
		    			return false;
		    		}
		    	})
			},
			cancelTask(){
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
						if(vm.oper_type == 'add'){
							eventBus.$emit('hide-task')
						}else{
							eventBus.$emit('hide-edit')
						}
					}).catch(() => {
						
					})
				}else{
					if(vm.oper_type == 'add'){
						eventBus.$emit('hide-task')
					}else{
						eventBus.$emit('hide-edit')
					}
				}
			},
			modifyTask(task_id,type){
				var vm = this;
				vm.oper_type = type;
				vm.task_id = task_id;
				axios.post('${ctx}/cell/1588License/getTaskInfoById.action',stringify({
					task_id : task_id,
					timeZone : timeZone
				})).then(function(response){
					var data = response.data;
					vm.ruleForm.task_name = data.task_name;
					if(vm.oper_type == 'view'){
						vm.fileUrl = '${ctx}/cell/1588License/getFileInfoById.action?file_id='+ data.file_id + '&time_zone=' + timeZone
					}
					setTimeout(function(){
						var fileRows = vm.$refs.ctable.getData();
						fileRows.map(function(item,index){
							if(item.file_id == data.file_id){
								vm.$refs.ctable.setCurrentRow(item);
								vm.clickFile(item);
							}
						})
					},100)
					vm.$nextTick(function(){
						vm.rightUrl = '${ctx}/cell/1588License/getCellRecordList.action?task_id=' + task_id;
    		    	});
					vm.ruleForm.file = data.file_id;
					vm.ruleForm.status = data.task_status;
					if(data.task_status == 'timing'){
						vm.ruleForm.exetime = data.start_time;
					}
					vm.defaultChecked = [data.file_id+''];
					vm.defaultTaskName = data.task_name;
					if(vm.oper_type == 'view'){
						vm.viewTaskFlag = false;
						vm.statusDisabled = true;
						setTimeout(function(){
							vm.setTimeEnable = true;
						},20)
						vm.taskNameDisabled = true;
					}else{
						vm.viewTaskFlag = true;
						vm.statusDisabled = false;
					}
					setTimeout(function(){
    					initForm(vm.$refs.ruleForm);
    				},500);
				})
			}
		},
		watch:{
			"ruleForm.status":function(newVal){
				if(newVal == 'timing'){
					this.setTimeEnable = false
				}else{
					this.setTimeEnable = true
				}
				this.$refs.ruleForm.validateField('exetime');
			},
			fileData(newVal){
				let mac_range = newVal.mac_range;
				this.queryParams.mac_range = mac_range;
				this.leftUrl = '${ctx}/cell/1588License/getEnbList.action'
				this.ruleForm.file = newVal.file_id;
			}
		},
		mounted(){
			eventBus.$off('save-task').$on('save-task',this.submit);
			eventBus.$off('save-edit-task').$on('save-edit-task',this.submit);
			eventBus.$off('cancel-task').$on('cancel-task',this.cancelTask)
			eventBus.$off('cancel-edit').$on('cancel-edit',this.cancelTask);
			eventBus.$off('get-info').$on('get-info',this.modifyTask)
		}
	})
</script>