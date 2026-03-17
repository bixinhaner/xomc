<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.deviceItem{
		display:inline-block;
	}
</style>
<div id="addDeviceGroupPage">
<div class="slideBody" style="background:#FFF;">
	<el-form :model='groupForm' :rules="groupRules" ref="groupForm" label-position="top">
		<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupName'>
			<el-input :disabled="disableFlag" v-model="groupForm.groupName" style='width:440px;'></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("MiaoShu")%>' prop='description'>
			<el-input type="textarea" maxlength=50 v-model="groupForm.description" style='width:440px;' :disabled="disableFlag"></el-input>
		</el-form-item>
		    <!--这里是UPS 的增加列表 2020/6/9 与杜红彦确定 暂时没有此功能 后期做的时候可放开-->
			<!--<el-pairgrid :readonly="disableFlag" :id="pairgridName" v-if='showPairGrid' ref="cpairgrid" @selection-change='selectChange'  
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
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1}" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop='serial_number' label='<%=rb.getString("DianYuanBianMa")%>'></el-table-column>
						<el-table-column  prop='device_group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
					
						
					</template>
					<template slot='toolbar'>
						<el-form :model='queryForm' ref="queryForm" label-position="top">
							<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
							:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
								<template slot="form">
									<el-form-item class='deviceItem' label='<%=rb.getString("DianYuanBianMa")%>' prop='serial_number'>
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
						<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'true'"  prop='device_group_name' label='<%=rb.getString("SheBeiZuMingCheng")%>'></el-table-column>
						<el-table-column v-if="addTypeFlag == 'false'" prop='macaddress' width="" label='MAC Address'></el-table-column>
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
			</el-form-item>-->
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
			}else{
				var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5]{1,50}$/;
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
			params_device:{
				group_id:''
			},
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
				like_fields:'serial_number'
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
		init(type,id,addTypeFlag){
			var vm = this;
			vm.$refs.groupForm.resetFields();
			vm.type = type;
			vm.group_id = id;
            var parVm = Vue.getInstance("#upsRegister")
			vm.groupOptions = parVm.$refs.ctableGroup.getData();
			//判断是enb 还是Cpe
			vm.pairgridName = 'enb_select_list';
				vm.leftUrl = '${ctx}/cell/cpeinfos/getEnbList.action'; //${ctx}/system/device/enodeb/queryENBInfoPageList.action
				vm.deviceTitle = ['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuanZeJiZhan")%>'];
				if(type == 'add'){
					vm.disableFlag = false;
				}
				if ( type != 'add'){
					if(type == 'info'){
						vm.buttonFlag = false;
					}else{
						vm.buttonFlag = true;
					}
					//判断type 类型 区分 查看和修改 查看不允许修改
					vm.rightUrl = "${ctx}/system/device/enodeb/queryENBInfoPageList.action?group_id="+id;
					if(type == "info"){
						vm.disableFlag = true;
					}else{
						vm.disableFlag = false;
					}
					axios.post('${ctx}/system/deviceGroup/findDeviceGroupInfo.action',stringify({
						id:id,
						timeZone:vm.timeZone
					})).then(function(response){
						let data = response.data;
						vm.groupForm.groupName = data.group_name;
						vm.groupForm.description = data.description;
					}).catch(function(error){})
				}
			
		},
		selectChange(selection){
				this.selection = selection
				var cellCodes = '';
				if(selection.length != 0){
					selection.map(function(item){
						cellCodes += item.small_cell_code + ","
					})
				}
				this.groupForm.cellCodes = cellCodes;
				
				var idsArr = selection;
				let idsStr = ''
				idsArr.map((item,index) => {
					idsStr += item.small_cell_code + '_' + item.product + ',';
					return idsStr
				});
				this.ids = idsStr.substring(0,idsStr.length-1)
		
			
		},
		query(val){
			this.resetQuery();
			this.queryForm.search_text = val;
			this.$refs.cpairgrid.reload();
		},
		advanceQuery(){
			this.queryForm.search_text = "";
			this.queryForm.group_id = this.queryForm_group_id;
			this.queryForm.serial_number = this.serialNumberSearch
			this.$refs.cpairgrid.reload();
		},
		resetQuery(){
			this.$refs.queryForm.resetFields();
		},
		closeAddGroup(){
			eventBus.$emit('close-dialog');
			deviceVue = {};
		},
		addDeviceGroup(){
			var param={}  , vm = this , url;
				param.name = vm.groupForm.groupName;
				param.desc = vm.groupForm.description;
				param.ids = vm.ids;
				if (vm.type == "add") {
					url = "${ctx}/system/deviceGroup/addAndAssignGroup.action";
				} else {
					url = "${ctx}/system/deviceGroup/modEnbDeviceGroup.action";
					param.group_id = vm.group_id
				}
		    	vm.$refs.groupForm.validate((valid) => {
		    		if(valid){
		    			axios.post(url,stringify(param)).then(function(response){
		    				let data = response.data;
		    				if ( data.success ){
		    					vm.$message.success(data.message)
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
	},
	mounted(){
		//this.init()
		eventBus.$off('open-dialog').$on('open-dialog',this.init);
	}
	
});

</script>