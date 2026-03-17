<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.deviceItem{
		display:inline-block;
	}
	.el-table-column--selection.cell{
}
.el-table-column--selection .cell{
	min-width: 0px !important

}
</style>
<%-- 窗口-添加、修改设备组 --%>
<%-- <div id="winAddOrModDeviceGroup" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<div class="itemDiv" style="height: 45px;">
				<span ><%=rb.getString("SheBeiZuMingCheng")%></span>
				<input type="text" name="deviceGroup_name" onblur="validateByRegex(event);" 
					must="1" vali-regex="/^[a-zA-Z0-9_\u4e00-\u9fa5]{1,50}$/" 
					err_prompt_id="deviceGroupNameFmtPrompt"
					class="border border-box item">
				<span style="display: inline-block;padding-left: 185px;">
					<span id="deviceGroupNameFmtPrompt" style="display:none;color:#E03030;"><%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%></span>	
				</span>			
			</div>
			<div class="itemDiv" style="margin-top: 20px;">
				<span style="vertical-align: top"><%=rb.getString("MiaoShu")%></span>
				<textarea rows="4" cols="20" name="deviceGroup_desc" maxlength=200 class="border border-box item" style="width: 250px; height: 100px; resize: none;"></textarea>
			</div>
		</div>
		<div region="south" data-options="border:false,height:57" >
			<div class="windowButtonGroup" style="margin-right:24px;">
				<a href="#" class="linkbutton linkbutton_trend" onclick="saveDeviceGroup()"><span><%=rb.getString("QueDing")%></span></a>
				<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
		</div>
	</div>
</div> --%>
<div id="addDeviceGroupPage">
<div class="slideBody" style="background:#FFF;">
	<el-form :model='groupForm' :rules="groupRules" ref="groupForm" label-position="top">
		<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupName'>
			<el-input :disabled="disableFlag" v-model="groupForm.groupName" style='width:440px;'></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("MiaoShu")%>' prop='description'>
			<el-input type="textarea" maxlength=50 v-model="groupForm.description" style='width:440px;' :disabled="disableFlag"></el-input>
		</el-form-item>
		
			<el-pairgrid :readonly="disableFlag" :id="pairgridName" v-if='showPairGrid' ref="cpairgrid" @selection-change='selectChange'  
				:right-url="rightUrl" :left-url="leftUrl" :height="height" :row-key="groupRowKey" :query-params="queryForm" :title="deviceTitle" 
				:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
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
						<el-table-column v-if="addTypeFlag == 'true'" prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'false'" prop='serial_number' label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'true'" prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
					
						<el-table-column v-if="addTypeFlag == 'false'" prop='macaddress' label='<%=rb.getString("MACDiZhi")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'false'" prop='imsi' label='IMSI'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'false'" prop='device_group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-form :model='queryForm' ref="queryForm" label-position="top">
							<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="queryPlaceholder"
							:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
								<template slot="form">
									<el-form-item class='deviceItem' :label='queryDeviceLabel' prop='serial_number'>
										<el-input v-model='serialNumberSearch'  size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' style='margin-left:40px;' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
										<el-select v-model="queryForm_group_id" size="mini">
											<el-option v-for="item in groupOptions" :key="item.id" :label="item.group_name" :value="item.id">
											</el-option>
										</el-select>
									</el-form-item>
								</template>
							</el-query>
						</el-form>
					</template>
					<template slot='right'>
						<el-table-column v-if="addTypeFlag == 'true'" prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'true'"  prop='group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'false'" prop='serial_number' label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'false'" prop='macaddress' width="" label='<%=rb.getString("MACDiZhi")%>'></el-table-column>
				</template>
			</el-pairgrid>
			<div v-else>
				<label style='font-size:16px;'><%=rb.getString("YiXuanZeJiZhan")%></label>
				<el-ctable :id="'selected_device_list'" ref="stable"  :url="rightUrl" :height="height" pagination="true" :query-params="fileParams">
					<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
					<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
				</el-ctable>
			</div>
			<el-form-item prop='cellCodes'>
				<el-input v-model='groupForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</el-form>
</div>
<div v-if="buttonFlag" class="slideFooter">
	<el-button type="primary" @click="addDeviceGroup"><%=rb.getString("QueDing")%></el-button>
	<el-button @click="closeAddGroup"><%=rb.getString("QuXiao")%></el-button>
</div>
</div>


<script>

var addDeviceGroupVue = new Vue({
	el:'#addDeviceGroupPage',
	data(){
		var validateGroupName = (rule,value,callback) => {
			var vm = this;
			if(value.trim() == vm.defaultTaskName){
				callback();
			}else if(value.trim() == '' || value.trim() == null){
				callback('<%=rb.getString("SheBeiZuMingChengTiShi")%>')
			}else{
				var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5\s]{1,50}$/;
				if(reg.test(value)){
					callback()
				}else{
					callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
				}
			}
		};
		return{
			pairgridName:'enb_select_list',
			buttonFlag:true,
			queryForm_group_id:'',
			addTypeFlag:'',//根据这个参数判断是添加设备 还是添加cpe
			disableFlag:false,
			height:'200px',
			menusGroup:[],
			deviceUrl:'',
			serialNumber:'<%=rb.getString("XiaoZhanBianMa")%>',
			searchText:'',
			groupData:[],
			enbFlag : '${ deviceType == 'eNB' }',
			cpeFlag : '${ deviceType == 'CPE' }',
			params_device:{
				group_id:''
			},
			queryDeviceLabel:'',
			queryPlaceholder:'',
			serialNumberSearch:'',
			showWindowInfo:false,
			dialogTitle:'',
			dialogUrl:'',
			menusDevices:[],
			rowDataGroup:[],
			rowDataDevice:[],
			showPairGrid:true,
			leftUrl:'',
			rightUrl:'',
			groupOptions:[],
			deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			groupForm:{
				groupName:'',
				description:'',
				cellCodes:'',
			},
			groupRules:{
				groupName:[
					{required:true,max:50,message:'<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>',trigger:'blur'},
					{validator:validateGroupName,trigger:'blur'}
				],
				description:[
					{max:100,trigger:'blur'}
				]
			},
			queryForm:{
				search_text:'',
				group_id:'',
				like_fields:'serial_number,macaddress'
			},
			type:'',
			groupId:'',
			timeZone:timeZone,
			ids:''
		}
	},
	computed: {
		groupRowKey() {
			var vm = this;

			return vm.addTypeFlag == 'true' ? 'serial_number':'macaddress';
		}
	},
	methods:{
		//type: add device group,  modify device group, info device group,
		//id:设备组的对应id 
		//addTypeFlagvm.enbFlag :  标识 CPE-false or eNB-true	
		init(type,id,addTypeFlag){
			var vm = this;
			vm.$refs.groupForm.resetFields();
			vm.type = type;
			vm.group_id = id;
			vm.addTypeFlag = addTypeFlag;
			
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			//判断是enb 还是Cpe
			if(vm.addTypeFlag  == 'true'){
				
				vm.queryDeviceLabel="<%=rb.getString("XiaoZhanBianMa")%>";
				vm.queryPlaceholder="<%=rb.getString("XiaoZhanBianMa")%>";
				
				vm.pairgridName = 'enb_select_list';
				vm.leftUrl = '${ctx}/cell/cpeinfos/getEnbList.action'; //${ctx}/system/device/enodeb/queryENBInfoPageList.action
				vm.deviceTitle = ['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuanZeJiZhan")%>'];
				
				//新建设备组
				if(type == 'add'){
					vm.buttonFlag = true;
					vm.disableFlag = false;
				}
				
				//设备组：info-详情 ,modify-修改
				if ( type != 'add'){
					if(type == 'info'){	
						// 隐藏底部新建，取消按钮，输入项，表格勾选置灰
						vm.buttonFlag = false;
						vm.disableFlag = true;
					}else{
						vm.buttonFlag = true;
						vm.disableFlag = false;
					}
					
					vm.rightUrl = "${ctx}/system/device/enodeb/queryENBInfoPageList.action?group_id="+id;

					axios.post('${ctx}/system/deviceGroup/findDeviceGroupInfo.action',stringify({
						id:id,
						timeZone:vm.timeZone
					})).then(function(response){
						let data = response.data;
						vm.groupForm.groupName = data.group_name;
						vm.groupForm.description = data.description;
					}).catch(function(error){})
				}
			}else{
				//CPE
				vm.queryDeviceLabel="<%=rb.getString("CPEBianMa")%>";
				vm.queryPlaceholder="<%=rb.getString("CPEBianMa")%>/<%=rb.getString("MACDiZhi")%>";
				vm.pairgridName = 'cpe_select_list';
				vm.leftUrl = '${ctx}/system/device/cpe/queryCPEInfoPageList.action';
				vm.deviceTitle = ['CPEs','<%=rb.getString("YiXuanSheBei")%>'];
				
				if(type == 'add'){
					vm.buttonFlag = true;
					vm.disableFlag = false;
				}
				
				//判断type 类型 区分 查看和修改 查看不允许修改	
				if ( type != 'add'){					
					if(type == 'info'){						
						vm.buttonFlag = false;
						vm.disableFlag = true;
					}else{
						//modify
						vm.buttonFlag = true;
						vm.disableFlag = false;
					}
								
					vm.rightUrl = "${ctx}/system/device/cpe/queryCPEInfoPageList.action?group_id="+id;
					
					//回显 设备组名称，描述
					axios.post('${ctx}/system/deviceGroup/findDeviceGroupInfo.action',stringify({
						id:id,
						timeZone:vm.timeZone
					})).then(function(response){
						let data = response.data;
						vm.groupForm.groupName = data.group_name;
						vm.groupForm.description = data.description;
					}).catch(function(error){})
				}
			}
			
		},
		
		/**
		* 新建，修改，详情设备组， 基站列表选中
		* @param selection{Array}   选中数据
		*/
		selectChange(selection){
			var vm = this;
			setTimeout(function(){
				if(vm.addTypeFlag  == 'true'){
					var sRows = vm.$refs.cpairgrid.getData(),
						codes = sRows.map(function(item){ return item.small_cell_code;}),
						idsStr = sRows.map(function(item){ return item.small_cell_code + '_' + item.product;});
					
					vm.groupForm.cellCodes = codes.join(',');
					
					vm.ids = idsStr.join(',')
				}else{
					var sRows = vm.$refs.cpairgrid.getData(), //选择行数据
						codes = sRows.map(function(item){ return item.cpe_code;});// 表格的 cpe_code
					
					vm.groupForm.cellCodes = codes.join(',');
					
					vm.ids = codes.join(',');
				}
			},50);
			
		},
		query(val){
			this.resetQuery();
			this.queryForm.search_text = val;
			//this.$refs.cpairgrid.reload();
		},
		advanceQuery(){
			this.queryForm.search_text = "";
			this.queryForm.group_id = this.queryForm_group_id;
			this.queryForm.serial_number = this.serialNumberSearch
			this.$refs.cpairgrid.reload();
		},
		resetQuery(){
			this.serialNumberSearch = '';
			this.queryForm_group_id = '';
		},
		closeAddGroup(){
			eventBus.$emit('close-dialog');
			addDeviceGroupVue = {};
		},
		addDeviceGroup(){
			var param={}  , vm = this , url;
		
			
			if(this.addTypeFlag  == 'true'){
				param.name = vm.groupForm.groupName;
				param.desc = vm.groupForm.description;
				param.ids = vm.ids;
				if (vm.type == "add") {
					param.pid = vm.group_id;
					url = "${ctx}/system/deviceGroup/addAndAssignGroup.action";
				} else {
					url = "${ctx}/system/deviceGroup/modEnbDeviceGroup.action";
					param.group_id = vm.group_id;
				}
				
		    	vm.$refs.groupForm.validate((valid) => {
		    		if(valid){
		    			axios.post(url,stringify(param)).then(function(response){
		    				let data = response.data;
		    				if ( data.success ){
		    					vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
								eventBus.$emit('query-group',deviceVue.queryGroupSearchText);
		    				}else {
		    					vm.$message.error(data.message)
		    				}
		    				vm.closeAddGroup();
		    				
		    			}).catch(function(error){})
		    		}else{
		    			return false;
		    		}
		    	})
			}else{
				param.name = vm.groupForm.groupName;
				param.desc = vm.groupForm.description;
				param.cpeCodes = vm.ids;
				if (vm.type == "add") {
					param.pid = vm.group_id;
					url = "${ctx}/cell/CPE/addDeviceGroup.action";
				} else {
					url = "${ctx}/cell/CPE/modDeviceGroup.action";
					param.group_id = vm.group_id
				}
				
		    	vm.$refs.groupForm.validate((valid) => {
		    		if(valid){
		    			axios.post(url,stringify(param)).then(function(response){
		    				let data = response.data;
		    				if ( data.success ){
		    					vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
								eventBus.$emit('query-group',deviceVue.queryGroupSearchText);
		    				}else {
		    					vm.$message.error(data.message)
		    				}
		    				vm.closeAddGroup();
		    				
		    			}).catch(function(error){})
		    		}else{
		    			return false;
		    		}
		    	})
				
			}
			
		}
	},
	mounted(){
		eventBus.$off('addModifyInfo-dialog').$on('addModifyInfo-dialog',this.init);
	}
	
});

</script>