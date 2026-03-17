<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<div id="selectGroupPage">
	<el-ctable ref="ctableGroup" :url="groupUrl" :height="height" :row-key="'id'" :query-params="params_device"
		pagination="true" :rownumber=true >
		<!-- 模糊查询 -->
		<template slot="toolbar">
			<!-- <div class="queryGroup">
				<el-input class='pairgrid-query' v-model="searchText" @keyup.enter.native="searchResult"
					:placeholder="serialNumber"></el-input>
		    	<i @click='searchResult' class="el-icon-search" style="margin-left: 10px;"></i>
			</div> -->
		</template>
		
		<el-table-column label='' width="30" prop="">
			<template slot-scope="scope">
           		<el-radio v-model="groupId" :label="scope.row.id">&nbsp;</el-radio>
         	</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150" prop="group_name"></el-table-column>
	</el-ctable>
	<div slot="tip" class="el-upload__tip" v-show="deviceGroupTip"><%=rb.getString("QingXuanZeSheBeiZu")%></div>
	<div style="padding-top:10px;">
		<el-button type="primary" @click="moveDevice"><%=rb.getString("QueDing")%></el-button>
		<el-button @click="closeMoveDevice"><%=rb.getString("QuXiao")%></el-button>
	</div>
</div>

<script>

var moveDeviceVue = new Vue({
	el:'#selectGroupPage',
	data:{
		height:'400px',
		enodebForm:{
			serialnumber:'',
			groupName:'',
		},
		groupUrl:'',
		groupId:'',
		serialNumber:'<%=rb.getString("XiaoZhanBianMa")%>',
		searchText:'',
		params_device:{
			group_id:''
		},
		codes:'',
		enbFlag:'',
		deviceGroupTip:false
	},
	methods:{
		//单个移动设备组参数
		init(codes,id,eNBFlag){
			var vm = this;
			vm.codes = codes
			vm.groupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?no_group_id=' + id;
			vm.enbFlag = eNBFlag;
		},
		//批量移动设备组参数
		inita(codes,id,eNBFlag){
			var vm = this;
			vm.codes = codes;
			vm.groupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?no_group_id=' + id;
			vm.enbFlag = eNBFlag;
		},
		closeMoveDevice(){
			eventBus.$emit('close-dialog');
		},

		moveDevice(){
			var vm = this , ids, url = "", params = {};
			
			params.toGroupId = vm.groupId
			
			if(vm.enbFlag == "true"){
				params.ids = vm.codes
				url = '${ctx}/system/deviceGroup/moveCellToDeviceGroup.action'
			}else{
				params.cpeCodes = vm.codes
				url = '${ctx}/cell/CPE/moveCpeToDeviceGroup.action'
			}
			if ( vm.groupId ){
				vm.deviceGroupTip = false;
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if(data.success){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});	
						vm.closeMoveDevice();
					}else{
						vm.$message({
							type:'error',
							message:data.message
						})
					}
					
				}).catch(function(error){}) 
				
			}else{
				vm.deviceGroupTip = true;
			}
		}
	},
	mounted(){
		eventBus.$off('open-dialog').$on('open-dialog',this.init);
		eventBus.$off('open-dialog-moveto').$on('open-dialog-moveto',this.inita);
		
	}
	
});

</script>




<%-- 

窗口-选择移动到的设备组
<div id="winSelectGroupToMove" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<table class="easyui-datagrid" id="gridSelectGroupToMove"
					data-options="fit:true,rownumbers:true,fitColumns:true,border:false,striped:true,singleSelect:true,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'id',hidden:true"></th>
						<th data-options="field:'built_in',hidden:true"></th>
						<th data-options="field:'group_name'" width="100"><%=rb.getString("SheBeiZuMingCheng")%></th>
					</tr>
				</thead>
			</table>
		</div>
		<div region="south" data-options="border:false,height:57">
			<div class="windowButtonGroup" style="margin-right:20px">
				<a href="#" class="linkbutton linkbutton_trend"  onclick="moveToGroup()"><span><%=rb.getString("QueDing")%></span></a>
				<a href="#" class="linkbutton linkbutton_nowanna"  onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>   
			</div>
		</div>
	</div>
</div> --%>