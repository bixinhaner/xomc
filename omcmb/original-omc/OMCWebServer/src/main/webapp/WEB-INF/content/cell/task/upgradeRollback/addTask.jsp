<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#addRollbackTask .el-form-item__label{
	line-height:26px;
	text-align:left;
}
#addRollbackTask .modeItem{
	margin-top:20px;
}
#addRollbackTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
.fileContent .el-radio{
	margin-right:30px;
}
.nameItem .el-form-item__error{
	margin-left:110px;
}
#addRollbackTask .pairgrid-left,#addRollbackTask .pairgrid-left .el-ctable{
	border:none;
	border-left:none;
}
#addRollbackTask .el-form-item,.addDeviceDialog .el-form-item{
	display: flex;
	align-items: center;
}
#addRollbackTask .el-form-item__content,.addDeviceDialog .el-form-item__content{
	margin-left: unset!important;
	width: 100%;
}
#addRollbackTask .deviceTableBoxCls{
	display:flex;
	height:372px;
	width:100%;
}
#addRollbackTask .deviceTableBoxCls .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
</style>

<!-- enb新建回退任务 -->
<div id="addRollbackTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" :hide-required-asterisk=true label-width="165px">
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item class="nameItem" label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' style='margin-left:45px;margin-top:18px;'>
			<el-input :disabled="showName" maxlength=100 v-model="ruleForm.taskname" size="mini" style="width:348px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div style='width:99%;height:1px;background:#E9E9E9;;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<div class='fileContent' style='margin-left:45px;width:93%;'>
			<el-form-item style='margin-top:18px;margin-bottom:20px' label="<%=rb.getString("ChangPinXingHao")%>" prop="product">
				<el-select v-model="ruleForm.product" :disabled="showName">
					<el-option v-for="item in productOptions" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' label-width="165px" style="margin-bottom: 10px;">
				<el-radio-group v-model="deviceType" :disabled="showName" style="padding-top: 5px;">
					<el-radio border label="all"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio border label="select"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<div class="deviceTableBoxCls">			
				<div style='border:1px solid #EFF0F2;flex:1;overflow:auto'>
					<el-pairgrid v-if="showDeviceType" :id="'select_device_list'" :rownumber="true" ref="add_task_table" :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="small_cell_code" :query-params="query_cell_params" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" @selection-change='selectChange' @right-load-success="rightLoadSuccess" :readonly="showName">
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
							<el-table-column prop='cell_name' label='<%=rb.getString("HostName")%>' width="220"></el-table-column>
							<el-table-column prop='cell_ip' label='IP' width="180"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' width="200"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' width="200"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHao")%>' width="150"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
						</template>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
								:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template slot="form">
										<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
											<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
											<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
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
										<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
											<el-select v-model="query_cell_form.software_version" size="mini">
												<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
											<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
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
					<el-ctable v-else ref="add_task_table_all" id="add_task_table_all" row-key="small_cell_code" :url="deviceUrl" :height="height" :query-params="query_cell_params" pagination="true">
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
						<el-table-column prop='cell_name' label='<%=rb.getString("HostName")%>' width="220"></el-table-column>
						<el-table-column prop='cell_ip' label='IP' width="180"></el-table-column>
						<el-table-column prop='rollback_version' label='Rollback Version' width="200"></el-table-column>
						<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' width="200"></el-table-column>
						<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHao")%>' width="200"></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
								:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template slot="form">
										<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
											<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
											<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
											<el-select v-model="query_cell_form.group_id" size="mini">
												<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='Rollback Version' prop='rollback_version'>
											<el-select v-model="query_cell_form.rollback_version" size="mini">
												<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
											<el-select v-model="query_cell_form.software_version" size="mini">
												<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
											<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
										</el-form-item>
									</template>
								</el-query>
							</el-form>
						</template>
					</el-ctable>
				</div>
			</div>
			<el-form-item prop='cellCodes' style='margin-bottom:20px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div style='width:99%;height:1px;background:#E9E9E9;;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;margin-left:45px;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status" :disabled="showName">
				<el-radio border label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio border label="suspend"><%=rb.getString("GuaQi")%></el-radio>
				<el-radio border label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;margin-bottom:32px;' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
	</el-form>
</div>

<script type="text/javascript">
var addRollback = new Vue({
	el:'#addRollbackTask',
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
					taskType:"2"
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
		var validateCodes = (rule,value,callback) => {
			if(this.deviceType == 'all'){
				callback()
			}else{
				if(value == '' || value==null){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		}
		return {
			leftUrl:'',
			rightUrl:'',
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			height:'370px',
			width:'90%',
			setTimeEnable:true,
			rowData : [],
			ruleForm:{
				taskname:'${addTaskName}',
				cellCodes:'',
				status:'active',
				exetime:'',
				product:""
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
			defaultTaskName:'',
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			deviceUrl:'',
			query_cell_params:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				module_type:'',
				productValue:'',
				rollback_version:''
			},
			query_cell_form:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				module_type:'',
				rollback_version:''
			},
			groupOptions:[],
			versionOptions:[],
			rbVersionOptions:[],
			firstFlag:true,
			showName:false,
			productOptions:[],
			deviceType:'select',
			showDeviceType:true,
			selection:[]
		}
	},
	methods:{ 
		init:function(){
			var vm = this;
			//vm.productOptions = enbFileVue.productTypeList;
			//新建回退任务时，将 eu,ru 产品类型过滤掉
			vm.productOptions = enbFileVue.productTypeList.filter(function(item){
				return item.value != 'CR-B4860/RU' && item.value != 'CR-B4860/EU';
			});
			
			var reg = new RegExp('\\\\',"g");
			var value = enbFileVue.product_type.replace(reg,'');
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : value,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : value,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : value,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data;
			}).catch(function(error){})
			if(enbFileVue.operType == "modifyTask" ||　enbFileVue.operType == "viewTask"){
				axios.post('${ctx}/task/upgrade/getTask.action',stringify({
					taskId : enbFileVue.rowDataTask.TASK_ID,
					timeZone : timeZone
				})).then(function(response){
					var data = response.data;
					vm.ruleForm.taskname = data.TASK_NAME;
					vm.ruleForm.status = data.CREATE_STATUS;
					if(data.CREATE_STATUS == 'timing'){
						vm.ruleForm.exetime = data.CREATE_TIME;
					}
					vm.defaultProduct = vm.ruleForm.product;
					vm.defaultTaskName = data.TASK_NAME;
					vm.deviceType = data.selectAll == "true"?"all":"select";
					
					vm.ruleForm.product = data.PRODUCT_TYPE;
					vm.query_cell_params.productValue = data.PRODUCT_TYPE.includes("CR-B4860") ? data.PRODUCT_TYPE.substr(0,8) : data.PRODUCT_TYPE.replace(reg,'');
					vm.$nextTick(function(){
						vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1&isRollBack=1';
						vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1&isRollBack=1';
						vm.$refs.ruleForm.clearValidate();
						vm.rightUrl = '${ctx}/task/upgrade/getTaskSelectedList.action?taskId=' + enbFileVue.rowDataTask.TASK_ID;
						initForm(vm.$refs.ruleForm);
    		    	});
					if(enbFileVue.operType == "viewTask"){
						vm.showName = true;
					}
				})
			}else{
				var dlist = enbFileVue.cellData.filter(function(row){
						return !['sBS77420','sBS77410','sBS77400','sBS77401'].includes(row.module_type);
					});

				vm.$refs.add_task_table.appendCheckedRows(dlist);
				vm.ruleForm.product = enbFileVue.product_type;
				vm.query_cell_params.productValue = enbFileVue.product_type.includes("CR-B4860") ? enbFileVue.product_type.substr(0,8) : enbFileVue.product_type.replace(reg,'');
				vm.$nextTick(function(){
					vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1&isRollBack=1';
					vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1&isRollBack=1';
					vm.$refs.ruleForm.clearValidate();
					setTimeout(function(){
						initForm(vm.$refs.ruleForm);
					},2000)
				});
			}
			vm.getQueryData(value);
		},
		getQueryData(product){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			var productVal = product.includes("CR-B4860") ? product.substr(0,8) : product;
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data;
			}).catch(function(error){})
		},
		query(val){
			this.query_cell_params.search_text = val;
		},
		advanceQuery(){
			this.query_cell_form.search_text = "";
			Object.assign(this.query_cell_params,this.query_cell_form);
		},
		resetQuery(){
			this.$refs.query_cell_form.resetFields();
		},
		submit(){
	    	var vm = this;
	    	var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			if(vm.deviceType == 'all'){
	    				params.selectAll = "true"
	    			}else{
	    				params.selectAll = "false"
	    				params.cellCodes = vm.ruleForm.cellCodes;
	    			}
	    			params.timeZone = timeZone;
	    			params.taskName = vm.ruleForm.taskname;
	    			params.status = vm.ruleForm.status;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.taskType = "2";
	    			var codes = {
	    					bu : 'CR-B4860/BU',
	    					eu : 'CR-B4860/EU',
	    					ru : 'CR-B4860/RU'
	    			}
	    			params.productValue = vm.ruleForm.product;
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
	    				url = '${ctx}/task/upgrade/addTask.action'
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
            // 防止多次提交
            if(enbFileVue.slideSubmitLoading)return

            enbFileVue.slideSubmitLoading = true;
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    enbFileVue.$refs.slide.hide();
                    enbFileVue.list_name = "rollback";
                    enbFileVue.$refs.upgrade_cell_table.clearSelection();
                    enbFileVue.$refs.rb_task_table.refresh();
				}else{
					vm.$message.error(data["message"]);
                    enbFileVue.slideSubmitLoading = false;
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
				}).catch(() => {})
			}else{
				enbFileVue.$refs.slide.hide();
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
		},
		selectChange(selection){
			this.selection = selection
		},
		rightLoadSuccess(){
			var vm = this;
			if(vm.firstFlag){
				if(enbFileVue.operType == "modifyTask"){
					var data = this.$refs.add_task_table.getData();
					var cellCodes = '';
					data.map(function(item){
						cellCodes += item.small_cell_code + ","
					})
					vm.ruleForm.cellCodes = cellCodes;
				}
				initForm(vm.$refs.ruleForm);
				vm.firstFlag = false;
			}
		}
	},
	watch:{
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
				this.reset
			}else{
				this.showDeviceType = true;
			}
			this.resetQuery();
		},
		"ruleForm.product":function(val){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			vm.query_cell_params.productValue = val.includes("CR-B4860") ? val.substr(0,8) : val.replace(reg,'');
			vm.getQueryData(val.replace(reg,''));
		},
		selection(val){ // 设置
			var vm = this,
				cellCodes = '';
			/*
			if(val.length != 0){
				val.map(function(item){
					cellCodes += item.small_cell_code + ","
				})
			}
			*/
			try{
				var list = vm.$refs.add_task_table.getData().map(function(item){
					return item.small_cell_code;
				});
				cellCodes = list.join(',');
			}catch(e){}

			this.ruleForm.cellCodes = cellCodes;
		},
	},
	mounted(){
		this.init();
		eventBus.$off('add-task').$on('add-task',this.submit);
		eventBus.$off('cancel-add-task').$on('cancel-add-task',this.cancel);
	}
})
</script>