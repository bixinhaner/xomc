<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
#addCpeUpgradeTask .el-form-item{
	margin-left:45px;
}
#addCpeUpgradeTask .el-pairgrid .el-form-item{
	margin-left:0;
}
.el-radio-group,.el-radio{
	line-height:50px;
}
.borderDiv{
	width:600px;
	height:50px;
	line-height:50px;
	border:1px solid #D1EcF5;
	padding-left:15px;
}
.el-table__row .selected-status{
	opacity:0;
}
.el-table__row.current-row .selected-status,.el-table__row:hover .selected-status{
	opacity:1;
}
.el-pairgrid .el-form-item{
	display:inline-block;
	margin-right:60px;
}
#addCpeUpgradeTask .tableDiv{
	height:20px;
	width:auto;
}
#addCpeUpgradeTask .el-radio{
	line-height: 14px;
}
.filterFile .el-checkbox__label {
	font-size:12px;
	font-weight:normal;
	margin-left:0;
}
.alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 20px; 
}
.titleStyML{
	margin-left: 20px;
}
</style>
<%-- 新建升级任务 --%>
<div id="addCpeUpgradeTask" ref="taskInfo" style="margin-top:20px;">
	<el-form :model="params" :rules="rules" ref="taskForm" label-position="left" >
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label="<%=rb.getString("RenWuMingCheng")%>" prop="taskName" label-width="120px" style="padding-top:14px;" >
			<el-input v-model="params.taskName" maxlength=100 style="width:616px;padding-top:7px;" :disabled="taskNameDisable"></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="productValue" label-width="120px" style="margin-bottom:10px;">
			<el-radio-group v-model="params.productValue" @change="productTypeSelect" :disabled="productTypeDisable" style="padding-top:-5px;">
				<el-radio class="CODE_CPE_UPGRADE_IMAGE hidden" label="1">ODU</el-radio>
				<el-radio class="CODE_CPE_UPGRADE_IMAGE hidden" :style="{marginLeft: marginLeft +'px'}" label="2">IDU</el-radio>
			</el-radio-group>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="divTitleStyle el-title-icon titleStyML">
			<span><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-form-item style="margin-bottom:0px;">
			<el-pairgrid :id="'cpe_select_device_list'" style='margin:10px 45px 10px 0px;' :height="height" v-if="pairgridShow" ref="pairgrid" :left-url="leftUrl" :right-url="rightUrl" :title="pairgridTitle" :messages="{placeholder:'<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%>'}"
				:row-key="'small_cell_code'" :query-name="'serial_number,host_name'"
				 @selection-change="selectChange" :query-params="queryParams" :rownumber="true">
				<template slot="left" pagination="true">
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
					<el-table-column prop="small_cell_code" v-if="false"></el-table-column>
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" width="250" ></el-table-column>
					<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" width="160" ></el-table-column>
					<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  width="160"></el-table-column>
					<el-table-column prop="model_name" sortable show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" width="160" ></el-table-column>
					<el-table-column prop="cell_name" label="<%=rb.getString("RSCellName")%>" width="160" sortable></el-table-column>
					<el-table-column prop="cell_identity" label="ECI" width="100" ></el-table-column>
					<el-table-column prop="pci" label="PCI" width="100" ></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" width="190" ></el-table-column>
				</template>
				<template slot="toolbar">
					<el-form :model='queryParams' ref="advanceForm" label-position="top">
						<el-query @query="query" @advance-query="advanceQuery" @reset="advanceReset" :ok-text="'<%=rb.getString("QueDing")%>'" 
							:reset-text="'<%=rb.getString("ChaXunChongZhi")%>'" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%>">
							<template slot="form">
								<el-form-item label='<%=rb.getString("CPEBianMa")%>' prop='serial_number'>
									<el-input v-model='queryParams.serial_number' size="mini"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("CPEName")%>' prop='host_name'>
									<el-input v-model='queryParams.host_name' size="mini"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
									<el-select v-model="queryParams.group_id" size="mini">
										<el-option v-for="item in deviceGroups" :label="item.text" :value="item.value">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("BanBen")%>' prop='software_version'>
									<el-select v-model="queryParams.software_version" size="mini">
										<el-option v-for="item in versions" :label="item.text" :value="item.value">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ChanPinXingHao")%>' prop='model_name'>
									<el-select v-model="queryParams.model_name" size="mini" filterable>
										<el-option v-for="item in modelName" :label="item.model_name" :value="item.model_name">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("HostName")%>' prop='cell_name'>
									<el-input v-model='queryParams.cell_name' size="mini"></el-input>
								</el-form-item>
						    </template>
						</el-query>
					</el-query>
				</template>
				<template slot="right">
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"></el-table-column>
					<%-- <el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" width="150" ></el-table-column>
					<el-table-column prop="software_version" label="<%=rb.getString("BanBen")%>" width="100"></el-table-column> --%>
				</template>
			</el-pairgrid>
			<div v-else>
				<label><%=rb.getString("YiXuanZeCPE")%></label>
				<el-ctable :id="'cpe_selected_device_list'" :url="rightUrl" :height="height" :pagination=true> 
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" width="200"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"></el-table-column>
					<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" width="200" ></el-table-column>
					<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" width="100"></el-table-column>
				</el-ctable>
			</div>
		</el-form-item>
		<el-form-item prop="cellCodes" style="margin-bottom:30px;">
			<el-input v-model="params.cellCodes" v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="divTitleStyle el-title-icon titleStyML">
			<span><%=rb.getString("WenJianLieBiao")%></span>
			<el-checkbox class="filterFile" v-model="noFilterFile" @change="filterFiles" label="<%=rb.getString("ZiDongGenJuMoKuaiXingHaoGuoLv")%>"></el-checkbox>
		</div>
		<el-form-item style="margin-bottom:0;">
			<div v-if="fileListShow">
				<el-ctable :id="'cpe_select_file_list'" style='margin:10px 45px 10px 0px;border:1px solid #E9E9E9;' :height="height" ref="fileList" :url="softVUrl" :query-params="FileParams" :pagination=true @row-click="rowClick"
					  @load-success='fileListLoadSuc' :default-checked="defaultChecked" :row-key="'id'">
					<el-table-column width="70" label="<%=rb.getString("XuanZe")%>">
						<template slot-scope="scope">
							<div class="tableDiv el-icon el-icon-status-yes selected-status" ></div>
						</template>
					</el-table-column>
					<el-table-column prop="id" v-if="false"></el-table-column>
					<el-table-column prop="file_name" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>"  width="300"></el-table-column>
					<el-table-column prop="version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" width="250"></el-table-column>
					<el-table-column prop="model_name" show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" width="160" ></el-table-column>
					<el-table-column prop="product" show-overflow-tooltip label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" width="120" :formatter="productFmt"></el-table-column>
					<el-table-column prop="size" label="<%=rb.getString("WenJianDaXiao")%>" width="150"></el-table-column>
					<el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" width="180"></el-table-column>
					<el-table-column prop="desc" show-overflow-tooltip label="<%=rb.getString("MiaoShu")%>"></el-table-column>
				</el-ctable>
			</div>
			<div v-else id="viewFileList">
				<el-ctable :height="height" style='margin:10px 45px 10px 0px;border:1px solid #E9E9E9;' ref="fileList" :url="softVUrl" :pagination=true
					  @load-success='fileListLoadSuc' :default-checked="defaultChecked" :row-key="'id'">
					<el-table-column width="70" label="<%=rb.getString("XuanZe")%>">
						<template slot-scope="scope">
							<div class="tableDiv el-icon el-icon-status-yes selected-status" ></div>
						</template>
					</el-table-column>
					<el-table-column prop="id" v-if="false"></el-table-column>
					<el-table-column prop="version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" width="150"></el-table-column>
					<el-table-column prop="product" show-overflow-tooltip label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" width="150"></el-table-column>
					<el-table-column prop="file_name" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>"  width="300"></el-table-column>
					<el-table-column prop="size" label="<%=rb.getString("WenJianDaXiao")%>" width="150"></el-table-column>
					<el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" width="180"></el-table-column>
					<el-table-column prop="model_name" show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" width="160" ></el-table-column>
					<el-table-column prop="desc" show-overflow-tooltip label="<%=rb.getString("MiaoShu")%>"></el-table-column>
				</el-ctable>
			</div>
		</el-form-item>
		<el-form-item prop="file_id" style="margin-bottom:30px;">
			<el-input v-model="params.file_id" v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="divTitleStyle el-title-icon">
			<span><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item prop="status" style="display:inline-block;">
			<el-radio-group v-model="params.status" @change="executedTypeSelect" :disabled="executeTypeDisable">
				<el-radio label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio v-show="false" label="suspend"><%=rb.getString("GuaQi")%></el-radio>
				<el-radio label="timing" style='margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop="time" style="display:inline-block;" :show-message="timeTipShow">
				<el-date-picker :disabled="timePickEnable" type="datetime" v-model="params.time" @focus="setTime" value-format="yyyy-MM-dd HH:mm:ss"
				  size="small" :picker-options="pickerOptions"></el-date-picker >
		</el-form-item>
	</el-form>
</div>


<script type="text/javascript">

/* $(function() { */
	closeLoading(); 
    new Vue({
    	el:"#addCpeUpgradeTask",
    	data(){
   			var vm = this;
    		var validateName = function(rule,value,callback){
    			if(value == ''){
    				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
    			}else{
    				if(value.trim() != vm.oldTaskName){
	    				axios.post('${ctx}/task/upgrade/cpe/taskNameExist.action',stringify({
	    					taskName:vm.params.taskName.trim()
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
    				}else{
    					callback();
    				}
    			}
    		};
    		return{
    			params:{
	    			taskName:"${addTaskName}", //任务名称
	    			productValue: writableMap['CODE_CPE_UPGRADE_IMAGE']?"1":"2", //产品类型
	    			cellCodes:"", //选中的设备
	    			file_id:"", //选中的软件版本
	    			file_name:"", //选中的软件版本
	    			status:"active", //执行方式
	    			time:"", //定时执行时间
   					timeZone:timeZone,
    				taskType: "1",
    				rawMode: false,
    				taskId:"",
    				version:''
    			},
    			rules:{
    				taskName:[
    					{validator: validateName,trigger:'blur'}
    				],
    				cellCodes:[
    					{required:true, message:'<%=rb.getString("QingXuanZeSheBei")%>'}
    				],
    				file_id:[
    					{required:true, message:'<%=rb.getString("QingXianXuanZeWenJian")%>'}
    				],
    				time:[
    					{required:false, message:'<%=rb.getString("QingXuanZeShiJian")%>'}
    				]
    			},
    			timePickEnable: true,//时间选择框是否可操作
    			pickerOptions:{
    				disabledDate(time){
    					return time.getTime()< Date.now()-8.64e7;
    				}
    			},
    			height:'370px',
    			marginLeft: writableMap['CODE_CPE_UPGRADE_IMAGE']? '30' :'0',
    			leftUrl: writableMap['CODE_CPE_UPGRADE_IMAGE']?"${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=1":"${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=2",
    			rightUrl:"",
    			softVUrl:"",
    			deviceGroups:[],//高级查询设备组选择下拉内容
    			versions:[],//高级查询版本选择下拉内容
				modelName:[], // 高级查询model下拉内容
    			queryParams:{
	    			serial_number :"", //高级查询基站编码
	    			host_name :"", //高级查询基站名称
	    			group_id :"", //高级查询选择的设备组
	    			software_version:"",//高级查询选中的版本 
					model_name:"",//高级查询选中的model 
					cell_name: ""
    			},
    			pairgridTitle:['<%=rb.getString("CPELieBiao")%>','<%=rb.getString("YiXuanZeCPE")%>'],
    			FileParams:{
   					timeZone:timeZone,
					model_name:''
    			},
    			timeTipShow : true,
    			defaultChecked:[],
    			operateType:'add',
    			oldTaskName:"",
    			oldProductValue :"",
    			selectDataChanges: [],
    			pairgridShow: true,
    			fileListShow: true,
    			taskNameDisable: false,
    			productTypeDisable: false,
    			executeTypeDisable: false,
    			taskStatus:"",
    			showFooter:true,
    			cancelTitle:'<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',
				selectData:'',
				isTitle:false,
				noFilterFile:true,
				selectedDevice:[]
    		}
    	},
    	watch:{
    		selectDataChanges(newVal,oldVal){
    			var data = this.$refs.pairgrid.getData();
    			var cellCodes = '';
    			if(data.length != 0){
    				data.map(function(item){
    					cellCodes += item.small_cell_code + ","
    				})
    			}
    			this.params.cellCodes = cellCodes
    		}
    	},
    	computed:{
		},
		created(){
			this.getAdvanceCntent();
			eventBus.$off('hander-ok').$on('hander-ok',this.submit);
			eventBus.$off('hander-cancel').$on('hander-cancel',this.cancelSubmit);
			eventBus.$off('row-modify').$on('row-modify',this.getTaskInfo);
			eventBus.$off('add-task').$on('add-task',this.addUpgradeTask);
			eventBus.$off('cancel-add').$on('cancel-add',this.cancelSubmit);
		},
    	methods:{
    		getAdvanceCntent(){//获取高级查询下拉列表内容
    			var vm = this;
    			axios.post('${ctx}/cell/CPE/getCpeSelectFilter.action',stringify({
    				forSelect:vm.params.productValue,
    				selectType:"deviceGroup"
    			})).then(function(response){
    				vm.deviceGroups = response.data;
    				vm.queryParams.group_id = vm.deviceGroups[0].value;
    			}).catch(function(error){});
    			axios.post('${ctx}/cell/CPE/getCpeSelectFilter.action',stringify({
    				forSelect:vm.params.productValue,
    				selectType:"version"
				})).then(function(response){
    				vm.versions = response.data;
    				vm.queryParams.software_version = vm.versions[0].value;
				}).catch(function(error){})
				axios.post('${ctx}/cell/CPE/queryModelNames.action',stringify({
    				forSelect:vm.params.productValue
				})).then(function(response){
    				vm.modelName = response.data;
    				vm.queryParams.model_name = vm.modelName[0].model_name;
				}).catch(function(error){})
			
    		},
    		addUpgradeTask(){
    			this.softVUrl = writableMap['CODE_CPE_UPGRADE_IMAGE']?'${ctx}/cell/version/queryfileInfosList.action?file_type=3&timeZone='+timeZone:"${ctx}/cell/version/queryfileInfosList.action?file_type=4&timeZone="+timeZone;
    		},
    		getTaskInfo(taskId,taskStatus){//修改任务，界面内容获取
    			this.showFooter = false;
    			this.cancelTitle = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
    			this.operateType = 'modify';
    			var vm = this; 
    			vm.taskStatus = taskStatus;
    			if(taskStatus != '1'){
    				vm.pairgridShow = false;
    				vm.fileListShow = false;
    				vm.taskNameDisable = true;
    				vm.productTypeDisable = true;
    				vm.executeTypeDisable = true;
    				vm.timePickEnable = false;
    			}
    			vm.rightUrl = "${ctx}/task/upgrade/cpe/getTaskSelectedList.action?taskId="+taskId+"&timeZone="+timeZone;
    			axios.post('${ctx}/task/upgrade/cpe/getTask.action',stringify({
    				taskId : taskId,
    				timeZone:timeZone
    			})).then(function(response){
    				let data = response.data;
    				vm.params.taskId = taskId;
    				vm.params.taskName = data.TASK_NAME;
    				vm.oldTaskName = data.TASK_NAME;
    				vm.params.productValue = data.PRODUCT;
    				vm.oldProductValue = data.PRODUCT;
    				vm.params.cellCodes = data.small_cell_code;
    				vm.params.file_id = data.FILE_ID;
    				if(data.PRODUCT == "1"){
        				vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=1";
        				if(taskStatus == '1'){
	        				vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=3"+"&timeZone="+timeZone;
        				}else{
	        				vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=3"+"&fileId="+data.FILE_ID+"&timeZone="+timeZone;
        				}
        			}else{
        				vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=2";
        				if(taskStatus == '1'){
	        				vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=4"+"&timeZone="+timeZone;
        				}else{
	        				vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=4"+"&fileId="+data.FILE_ID+"&timeZone="+timeZone;
        				}
        			}
    				vm.defaultChecked = [data.FILE_ID+''];
    				vm.params.status = data.CREATE_STATUS;
					vm.executedTypeSelect(data.CREATE_STATUS);
    				if(data.CREATE_STATUS == 'timing') vm.params.time = data.CREATE_TIME;
    				setTimeout(function(){
    					initForm(vm.$refs.taskForm);
    				},500);
    			}).catch(function(error){})
    		},
    		fileListLoadSuc(){//选中文件列表回显
    			var vm = this;
				var tb = vm.$refs.fileList;
				setTimeout(function(){
    				tb.tbData.map(function(row){
						if(row.id == vm.defaultChecked[0]) tb.setCurrentRow(row)
    				});
				},200);
    		},
    		submit(type){//修改新建保存按钮
    			var vm = this;
    			var url = "${ctx}/task/upgrade/cpe/addTask.action";
    			var message = "<%=rb.getString("ChengGong")%>";
    			if(type=="modify") {
    				url = "${ctx}/task/upgrade/cpe/updateTask.action";
        			if(!isFormChanged(vm.$refs.taskForm)){
						vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							type:'warning'
						}).then().catch();
						return;
   	    			}
    			}
   				vm.$refs.taskForm.validate(function(valid){
       				if(valid){
       					axios.post('${ctx}/task/upgrade/cpe/isNeedMoreTimeForUpgrade.action', stringify({
       						cpeCodes:vm.params.cellCodes,
       						destVersion:vm.params.version
       					})).then(function(response){
       						let data = response.data;
       						
       						if (data["isNeedMoreTime"]) {
       							vm.$confirm( "<%=rb.getString("ShengJiShiJianJiaoChangShiFouJiXu")%>" ,'<%=rb.getString("QueRen")%>',{
       			    				customClass:'warningConfirm',
       								confirmButtonText:'<%=rb.getString("QueDing")%>',
       								cancelButtonText:'<%=rb.getString("QuXiao")%>',
       								type:'warning',
       								closeOnClickModal:false
       							}).then(function(){
       								vm.saveTask(url,vm.params,message);
       							}).catch()
       						} else {
       							if(type == 'add'){
									   var msg = ''
									   if(vm.isTitle){
											msg = "<%=rb.getString("PiPeiXingHaoWenJianQueDing")%>" +"<%=rb.getString("QueRenXinJianRenWu")%>"
									   }else{
										   msg = "<%=rb.getString("QueRenXinJianRenWu")%>"
									   }
       								vm.$confirm( msg ,'<%=rb.getString("QueRen")%>',{
       				    				customClass:'warningConfirm',
       									confirmButtonText:'<%=rb.getString("QueDing")%>',
       									cancelButtonText:'<%=rb.getString("QuXiao")%>',
       									type:'warning',
       									closeOnClickModal:false
       								}).then(function(){
       									vm.saveTask(url,vm.params,message);	
       								}).catch()
       							}else{
       								vm.saveTask(url,vm.params,message);	
       							}
       						}
          				}).catch(function(error){
          				
          				})
       				}else{
       					return false;
       				}
       			})
    		},
    		saveTask(url,params,message){
    			var vm = this;
    			axios.post(url, stringify(params)).then(function(response){
  						let data = response.data;
  						if (data["success"]) {
       	   					vm.$message({
	    						message: message,
	    						type:'success',
	    					})
                            eventBus.$emit('reload');
                            eventBus.$emit('closeRightDiv','success');
                            eventBus.$emit('to-task-view');//调整到任务查看页面
  						} else {
  							vm.$alert(data["message"], '<%=rb.getString("TiShi")%>',{
  								confirmButtonText:'<%=rb.getString("QueDing")%>',
  								type:'error'
  							}).then().catch(function(){})
  						}
   				}).catch(function(error){
   				
   				})
    		},
    		cancelSubmit(){
    			var vm = this;
   				if(isFormChanged(this.$refs.taskForm)){
	    			this.$confirm( "<%=rb.getString("QueDingLiKaiDangQianYeMian")%>" ,'<%=rb.getString("QueRen")%>',{
	    				customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(
						function(){
							eventBus.$emit('closeRightDiv','success');
						}	
					).catch()
   				}else{
   					eventBus.$emit('closeRightDiv','success');
   				}
    		}, 
    		productTypeSelect(label){//IDU ODU切换操作
				var vm = this;
    			vm.getAdvanceCntent();
				vm.$refs.pairgrid.clear();
   				vm.params.file_id = "";
   				vm.params.file_name = "";
   				vm.params.cellCodes = "";
   				if(vm.oldProductValue == label){
   					vm.$refs.pairgrid.reloadRightTb();
   					vm.fileListLoadSuc();
	   				vm.params.file_id = vm.defaultChecked[0];
   				}
    			if(label == "1"){
    				vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=1";
    				vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=3";
    			}else{
    				vm.leftUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=2";
    				vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=4";
    			}
    		},
    		executedTypeSelect(label){
    			if(label == "timing" && (this.taskStatus==""? true : this.taskStatus=='1')){
    				this.timePickEnable = false;
    				this.rules.time[0].required = true;
    				this.timeTipShow = true;
    			}else{
    				this.timePickEnable = true;
    				this.rules.time[0].required = false;
    				this.timeTipShow = false;
    			} 
    		},
    		setTime(){
    			this.params.time = formatDate(new Date(gloableTime));
    		},
    		query(val){
    			this.$refs.advanceForm.resetFields();
    			this.queryParams['searchText']= val;
				this.$refs.pairgrid.reload();
    		},
    		advanceQuery(){
				this.queryParams['searchText'] = "";
				this.$refs.pairgrid.reload();
    		},
    		advanceReset(){
    			this.$refs.advanceForm.resetFields();
    		},
    		rowClick(row, event, column){
				var vm = this;
    			vm.params.file_id = row.id;
    			vm.params.file_name = row.file_name;
    			vm.params.version = row.version;
				var model_name = row.model_name;
				var newStr = '';
				var newArr = '';
				var arr = [];
				if(vm.selectData!==''){
					newStr = vm.selectData.substring(0,vm.selectData.length-1)
					newArr = newStr.split(',')
					arr = newArr.filter(function(val){
						return val != model_name
					})
				}else{
					arr = []
				}
				
				if(arr.length>0 ){
					vm.isTitle = true // 等于true的时候数组里不止有指定的  弹出提示
				}else{
					vm.isTitle = false // 等于false的时候数字只有指定的 不弹出提示
				}
    		},
    		selectChange(selection,ele){
				var vm = this;
					modelName='';
					vm.selectDataChanges = selection;
					allSelect = '';
					vm.params.file_id ='';
    				vm.params.file_name = '';
    				vm.params.version = '';
    			vm.selectedDevice = selection;
    			
    			if (!vm.noFilterFile){
    				return
    			}
    			
					if(selection.length === 0){
						vm.selectData = ''
					}
					selection.map(function(item){
						 allSelect += item.model_name + ",";
						 vm.selectData = allSelect
						if(item.model_name !==null){
    						modelName += item.model_name + ","
						}
    				})
				if(selection != 0){
					if(selection.length === 1){
						if(vm.params.productValue === '1'){
							vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=3";
							vm.FileParams.model_name = selection[0].model_name
						}else{
							vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=4";
							vm.FileParams.model_name = selection[0].model_name
						}
					}else{
						if(vm.params.productValue === '1'){
							vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=3";
							vm.FileParams.model_name = modelName
						}else{
							vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=4";
							vm.FileParams.model_name = modelName
						}
					}
					
				}else{
						if(vm.params.productValue === '1'){
							vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=3";
							vm.FileParams.model_name = modelName
						}else{
							vm.softVUrl = "${ctx}/cell/version/queryfileInfosList.action?file_type=4";
							vm.FileParams.model_name = modelName
						}
					}
    		},
    		filterFiles(val){
    			var vm = this;
    			
    			if (val){
    				vm.selectChange(vm.selectedDevice);
    			}else {
    				vm.FileParams.model_name = "";
    				vm.$refs.fileList.setCurrentRow("");//清空已选择文件
    			}
    		},
    		productFmt(row,column,value,index){
    			if(value == 'CPE_VERSION' || value == "ODU"){
    				return "ODU";
    			}else if(value == 'CPE_IDU_VERSION' || value == "IDU"){
    				return "IDU";
    			}
    		}
    	}
    })
/*  }); */

</script>