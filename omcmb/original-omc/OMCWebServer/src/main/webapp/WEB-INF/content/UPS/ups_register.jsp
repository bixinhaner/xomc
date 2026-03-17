<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
	#upsRegister{
		display:flex;
	}
	.groupMgmt {
		width:300px;
		height: 100%;
		border-right:1px solid #E9E9E9;
		position:relative;
		display:flex;
		flex-direction:column;
	}
	.groupMgmt .circleIcon{
		top:10px
	}
	.deviceList {
		flex-grow:1;
		overflow:auto;
	}
	.groupMgmt .splitTitle{
		font-size:14px;
		font-weight:bold;
		color:#363B4E;
		display:inline-block;
		height:34px;
		line-height:34px;
		padding:10px
	}
	.el-icon-circle-info:before{
		color:#CFCFCF;
	}
	.el-icon-common-download:before{
		color:#363B4E;
	}
	#importDeviceCard .el-card__footer{
		border-top:none;
	}
	
	#upsRegister .el-upload__tip{
		color:red;
	}
	.ovhde{
		overflow: hidden
	}
	.curpo{
		cursor: pointer;
	}
	.deviceList .circleIcon{
		top:8px
	}
	.r110{
		right: 110px
	}
	.r60{
		right: 60px
	}
	.importBox{
		width:540px;
		height:227px;
		border:1px solid #E4E7EC;
		position:absolute;
		right:40px;
		top:50px;
	}
	.importBox-fileSlect{
		display:inline-block;
		height:20px;
		width:20px;
		margin-top:4px;
		cursor:pointer;
	}
	.importBox-file{
		float:right;
		font-size:16px;
	}
	.el-table .cell.el-tooltip{
		min-width: 0px !important
	}
	#upsRegister .treeItemBoxCls{
		width: 300px;
		position: relative;
	}
	#upsRegister .treeItemBoxCls .operCls{
		position:absolute;
		right:3px;
		z-index: 66;
	}
	.subGroupDeviceTitleCls{
		font-size: 12px;
		color: #333333;
		margin: 10px 0px;
	}
	.subGroupDeviceBoxCls .el-pairgrid-title{
		top:10px!important;
		right: 15px!important;
	}
	#upsRegister  .groupMgmt .el-input.el-input--small{
		width: 200px;
	}
	#upsRegister  .groupMgmt .el-tree-node__content{
		height: 30px;
	}
	#upsRegister .groupTreeBox{
		height: calc(100% - 44px);
		overflow: auto;
	}
	#upsRegister .ItemLabelCls{
		display: inline-block;
		width: 220px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>

<div id="upsRegister" class="panelDefault ovhde" >
	<!--左侧设备组-->
	<div class="groupMgmt">
		<!-- 按钮  添加设备组 -->
		<div class="circleIcon" >
			<span class="el-icon el-icon-circle-add CODE_UPS hidden" @click="addDeviceGroup"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<div class="splitTitle"><%=rb.getString("SheBeiZu")%></div>
		<div style="height: calc(100% - 54px);">
			<el-query type="normal" @query="queryGroupList" placeholder="<%=rb.getString("SheBeiZuMingCheng")%>" style="margin-bottom:10px;"></el-query>
			<div class="groupTreeBox">
				<el-tree 
					ref="groupTree"
					:data="groupData"
					node-key="id"
					:default-expanded-keys="defaultexpandedKeys"
					:default-checked-keys="defaultCheckedKeys"
					:props= "{label:'group_name'}"
					:highlight-current="true"
					@node-click="groupRowClick"
				>
					<div class="treeItemBoxCls" slot-scope="{ node,data }">
						<span class="ItemLabelCls" :title="node.label">{{node.label}}</span>
						<span v-if="optWriteShow || data.children" class="el-icon el-icon-operation-more operCls" @click="groupOpClick(node,data,event)" v-clickoutside="handerClose"></span>
					</div>
				</el-tree>
			</div>
			<el-cmenu ref="menuGroup" :data="menusGroup" @click="clickMenu"></el-cmenu>
		</div>
	</div>
	<div class="deviceList">
		<!-- 按钮   添加设备 -->
		<div v-if="optWriteShow" class="circleIcon r110" >
			<span class="el-icon el-icon-circle-add" @click="addDevice"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<!-- 按钮   导入 -->
		<div v-if="optWriteShow" class="circleIcon r60">
			<span class="el-icon el-icon-circle-import" @click="importDevice"></span>
			<div class="titleButtonText"><%=rb.getString("DaoRu")%></div>
		</div>
		
		<!-- 按钮   导出 -->
		<div class="circleIcon">
			<span class="el-icon el-icon-circle-export" @click="exportDevice"></span>
			<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
		</div>
		
		<el-ctable 
			id="egwDeviceTable"
			ref="ctableDevice" 
			:url="deviceUrl" 
			:height="height" 
			:row-key="'ups_code'" 
			:query-params="params_device"
			pagination="true" 
			:rownumber=true 
			@selection-change='batchSelect'
		 >
			
			<!-- 模糊查询 -->
			<template slot="toolbar">
				<div style="display: flex;align-items: center;">
					<div style="padding-left: 10px;">{{deviceTitle}}</div>
					<el-query type="normal" @query="searchResult" placeholder="<%=rb.getString("DianYuanBianMa")%>"></el-query>
				</div>
			</template>
			<!--设备列表-->
			<el-table-column width="50" type="selection" prop="ck" v-if="optWriteShow && groupWritable" key="upsck"></el-table-column>
			<el-table-column width="30" prop="" class-name="no-text-tips" v-if="optWriteShow" key="upsopts">
				<template slot-scope="scope">
            		<div class="el-icon el-icon-operation-more curpo" @click="optDeviceClick(scope.row,event)" v-clickoutside="handerClose" ></div>
          		</template>
			</el-table-column>
			<el-table-column prop="connection_status" width="50" sortable>
				<template slot-scope="scope">
					<div :class="{
						'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
						'':scope.row.have_connected==2,
						'conn_exc':scope.row.connection_status=='Exception',
						'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
				</template>
			</el-table-column>
			<el-table-column prop="serial_number" show-overflow-tooltip label='<%=rb.getString("DianYuanBianMa")%>'  sortable></el-table-column>
		</el-ctable>
		<el-cmenu ref="menuDevices" :data="menusDevices" @click="clickDeviceMenu"></el-cmenu>
	</div>
	<!--添加新一级设备组-->
	<el-dialog 
		:title='dialogTitle' 
		:visible.sync="showDeviceGroupDialog" 
		ref="windowDialog" 
		:width="deviceGroupDialogWidth"  
		:height='deviceGroupDialogHeight'
		:close-on-click-modal="false"  
		@close='closeDeviceGroupDialog'  
	 >
	 	<el-form  
		 	:model="groupForm"
			ref="deviceGroupDialogForm" 
			:rules="groupRules" 
			label-width="120" 
			label-position="left" 
			id="deviceGroupDialogForm" 
			:hide-required-asterisk='true'
		 >
			<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupName'>
				<el-input :disabled="viewDeviceGroupDialog" v-model="groupForm.groupName" style='width:200px;padding-top:7px;'></el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("MiaoShu")%>' prop='description' v-if="false">
				<el-input type="textarea" maxlength=50  style='width:440px;' :disabled="viewDeviceGroupDialog"></el-input>
			</el-form-item>
		</el-form>
		<span slot="footer" v-show="!viewDeviceGroupDialog">
			<div>
				<el-button type="primary" @click="addDeviceGroupSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeDeviceGroupDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
	<!--添加新二级设备组-->
	<el-dialog 
		:title='dialogTitle' 
		:visible.sync="showSubGroupDialog" 
		ref="subGroupDialog" 
		:width="subGroupDialogWidth"  
		:height='subGroupDialogHeight'
		:close-on-click-modal="false"  
		@close='closeSubGroupDialog'  
	 >
	 	<el-form  :model="subGroupForm" ref="subGroupDialogForm" :rules="subGroupRules"  label-position="left" label-width="120" id="subGroupDialogForm" :hide-required-asterisk='true'>
			<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupName'>
				<el-input :disabled="viewSubGroupDialog" v-model="subGroupForm.groupName" style='width:440px;padding-top:5px;'></el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("MiaoShu")%>' prop='description'>
				<el-input type="textarea" maxlength=50 v-model="subGroupForm.description" style='width:440px;padding-top:5px;' :disabled="viewSubGroupDialog"></el-input>
			</el-form-item>
			<div class="subGroupDeviceTitleCls">UPS</div>
			<div class="subGroupDeviceBoxCls">
				<el-pairgrid
					id="subGroupDevicePairgrid" 
					:rownumber="true" 
					ref="subGroupDevicePairgrid" 
					:right-url="deviceRightUrl" 
					:left-url="deviceLeftUrl" 
					height="240px" 
					:row-key="'ups_code'"
					:readonly="viewSubGroupDialog"
					:query-params="querySubGroupDeviceParams" 
					query-name="serial_number" 
					:title="deviceTableTitle" 
					:messages="{placeholder:'<%=rb.getString("DianYuanBianMa")%>'}" 
					@selection-change="devicesChange" 
					>
					<template slot="left">
						<el-table-column type='selection' width="50" align="center"></el-table-column>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("DianYuanBianMa")%>" min-width="150"></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="querySubGroupDeviceList" placeholder="<%=rb.getString("DianYuanBianMa")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("DianYuanBianMa")%>" min-width="150"></el-table-column>
					</template>
				</el-pairgrid>
			</div>
			<el-form-item label-width="0" prop="devices"></el-form-item>
		</el-form>
		<span slot="footer" v-show="!viewSubGroupDialog">
			<div>
				<el-button type="primary" @click="addSubGroupSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeSubGroupDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
	<!--设备 新增 修改 详情弹窗-->
	<el-dialog 
		:title='deviceDialogTitle'
		:visible.sync="showDeviceDialog" 
		ref="deviceDialog" 
		width="500"
		:close-on-click-modal="false"
		:append-to-body="true"
		@close="closeDeviceDialog"
	 >
		<el-form  :model="deviceDialogForm" ref="deviceDialogForm" :rules="deviceDialogRules" label-position="top" id="deviceDialogForm">
			<div>
				<p style="padding-bottom:20px;"><%=rb.getString("eNBZhuCeTiShiWenZi")%></p>
				<el-form-item label='<%=rb.getString("DianYuanBianMa")%>' prop='serialnumber'>
					<el-input type="textarea" v-model="deviceDialogForm.serialnumber"></el-input>
				</el-form-item>
				<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>'>
					<el-select v-model="deviceDialogForm.groupId" >
						<el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
					</el-select>
				</el-form-item>
			</div>
		</el-form>
		<span slot="footer">
			<div>
				<el-button type="primary" @click="addDeviceSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeDeviceDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
	<!-- 批量操作  -->
	<el-bulk target="egwDeviceTable" :list="selection" :row-key="'ups_code'" show-prop="serial_number"
		:message="bulkTableMessage">
		<template slot="button">
			<a class="linkbutton" @click="deleteCells"><span><%=rb.getString("PiLiangShanChu")%></span></a>
			<a class="linkbutton" @click="movecells"><span><%=rb.getString("YiDongDaoSheBeiZu")%></span></a>
		</template>
	</el-bulk>
	<!--移动设备到设备组-->
	<el-dialog :title='moveDevice.dialogTitle' :visible.sync="moveDevice.showMoveInfo" :width="moveDevice.windowWidth">
		<el-ctable ref="ctableGroup" :url="moveDevice.groupUrl" :height='moveDevice.height'>
		   	<el-table-column label='' width="30" prop="">
			<template slot-scope="scope">
           		<el-radio v-model="moveDevice.groupId" :label="scope.row.id">&nbsp;</el-radio>
         	</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150" prop="group_name"></el-table-column>
		</el-ctable>
		<div>
			<el-button type="primary" @click='moveTrue'><%=rb.getString("QueDing")%></el-button>
			<el-button @click="moveDevice.showMoveInfo = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- 删除设备组弹窗 -->
	<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="showDeviceGroupInfo" width="400" 
		:close-on-click-modal="false" @close="showDeviceGroupInfo = false">
		<div><%=rb.getString("QueDingShanChuSheBeiZu")%></div>
		<div v-show="delGroupType == 'stair'" style="font-size:12px;color:#999999;padding-top:8px;"><%=rb.getString("ZuNeiZiJiSheBeiZuJiSheBeiHuiBeiZiDongYiZhi")%></div>
		<div v-show="delGroupType == 'second'" style="font-size:12px;color:#999999;padding-top:8px;"><%=rb.getString("ZuNeiSheBeiHuiBeiZiDongYiDongDaoMoRenFenZu")%></div>
		<span slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="deleteDeviceGroup"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="showDeviceGroupInfo = false"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</span>
	</el-dialog>
	<!-- 导入文件框 -->
	<div>
		<transition name='el-zoom-in-top'>
			<el-card v-show='showImportCard' class='importBox' id='importDeviceCard'>
				<div slot='header'>
					<span>Import Device</span>
					<span class='el-icon el-icon-close importBox-file' style='' @click='closeImportDevice'></span>
				</div>
				<div>
					<label style='display:block;margin-bottom:5px;'>File</label>
						<el-upload :on-success='checkFile' :on-change="fileChange"  :show-file-list=false ref="upload" 
						     :action="uploadFileURL" :data="fileParams" name="uploadFile" :auto-upload="false">
							<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
								<a slot="suffix" class="el-icon el-icon-operation-import importBox-fileSlect" @click="fileSelect"></a>
							</el-input>
							<div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("ZhiZhiChiExeclFile")%></div>
							<div slot="tip" class="el-upload__tip" v-show="selectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
					
					<p style='color:#999;margin-top:10px;'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span>Please import a file as the format specified in the sample tempalte.
						<span class='curpo' @click="exportTemplate">
							<span style='vertical-align:top' class='el-icon el-icon-common-download'></span>
							<span style='color:#363B4E;text-decoration:underline'>Export Template</span>
						</span>
					</p>
					<el-button-group size="mini" style='margin-top:30px;'>
		    			<el-button type="primary" size="mini" @click="uploadDevice">OK</el-button>
		    			<el-button size="mini" @click="closeImportDevice">Cancel</el-button>
		    		</el-button-group>
				</div>
			</el-card>
		</transition>
	</div>
	<!-- 上传文件的用的表单 -->
	<form enctype="multipart/form-data" method="post" id="importDeviceForm">
	    <input name="fileSize"  value="" hidden="true">
	    <input name="operType" value="" hidden="true">
	    <input name="uploadFile"  id="importDeviceFile"  type="file" style="display: none;">
	</form>
	<!-- 下载模板用的表单 -->
	<form id="exportDeviceForm" style="display:none" method="post"></form>
</div>


<script>
var upsRegisterVue = new Vue({
	el:'#upsRegister',
	data(){
		var vm = this;

		var validateGroupName = (rule,value,callback) => {
				var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5,\s]{1,50}$/;
				
				if(value){
					if(reg.test(value)){
						callback()
					}else{
						callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
					}
				}else{
					callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
				}
			},
			validateDevice = function(rule,value,callback) { // 校验设备
				if(value.length == 0) {
					// callback('<%=rb.getString("QingXuanZeSheBei")%>');
					callback();
				}else {
					callback();
				}
			},
			validatorNum = (rule,value,callback) => {
				var serialNumber = value,
					temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
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
			// 设备弹窗 参数
			viewDeviceDialog:false,
			deviceDialogTitle:'',
			showDeviceDialog:false,
			deviceDialogForm:{
				serialnumber:'',
				groupId:''
			},
			deviceDialogRules:{
				serialnumber:[
					{validator:validatorNum,trigger:'blur'}
				],
			},
			moveDevice:{ // 移动设备对象
				groupUrl:'', //tabUrl
				showMoveInfo:false, // 移动设备弹窗
				height:'400px', // 表格高度
				dialogTitle:'', //弹窗title
				windowWidth:'', // 弹窗宽度
				groupId:'', // 设备组ID
				code:'', //设备组code值
			},
			selectFlag:false,        //标识是否选择了文件
			typeFlag:true,           //校验已选择的文件格式
			fileName:'',
			fileParams:{},            //上传文件时自定义的参数
			height:'100%',
			menusGroup:[],
			deviceUrl:'',
			params_device:{
				group_id:'',
				searchText:'',
				timeZone:timeZone,
			},
			showDeviceGroupDialog:false,
			dialogTitle:'',
			menusDevices:[],
			rowDataGroup:[],
			rowDataDevice:[],
			viewDeviceGroupDialog:false,
			groupForm:{
				groupName:'',
			},
			groupRules:{
				groupName:[
					{validator:validateGroupName,trigger:'blur'}
				],
				description:[
					{max:100,trigger:'blur'}
				]
			},
			deviceGroupDialogHeight:'600px',
			deviceGroupDialogWidth:'500px',
			subGroupForm:{
				groupName:'',
				description:'',
				devices:''
			},
			subGroupRules:{
				groupName:[
					{required:true,max:50,message:'<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>',trigger:'blur'},
					{validator:validateGroupName,trigger:'blur'}
				],
				description:[
					{max:100,trigger:'blur'}
				],
				devices:[
					{validator: validateDevice}
				],
			},
			deviceTableTitle:['','<%=rb.getString("YiXuanSheBei")%>'],
			deviceRightUrl:'',
			deviceLeftUrl:'${ctx}/ups/queryUpsInfosListByGroupId.action',
			querySubGroupDeviceParams:{
				searchText: '',
				timeZone: timeZone
			},
			showSubGroupDialog:false,
			viewSubGroupDialog:false,
			subGroupDialogHeight:'600px',
			subGroupDialogWidth:'500px',
			subGroupWindowType:'',
			modelHeight:'',
			groupWindowType:'',
			deviceTitle:'UPS',
			selectedDeviceShow: false,
			deviceGroupOptions:[],
			selection:'',
			showImportCard:false,
			deviceName: '',
			uploadFileURL: '${ctx}/ups/uploadFile.action',
			bulkTableMessage:{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("DianYuanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'},
			groupData:[],
			showDeviceGroupInfo:false,
			delGroupType:'',
			queryGroupSearchText:'',
			defaultexpandedKeys:[],
			defaultCheckedKeys:[]
		}
	},
	methods:{
		// 初始化
		init(){	
			var vm = this;
			vm.queryGroupList(vm.queryGroupSearchText);
		},
		// 设备组查询
		queryGroupList(val){
			var vm =this,
				params={
					search_text:val,
					isUps: 1,
					type: 3
				};
			vm.queryGroupSearchText = val;
			vm.defaultexpandedKeys =[];
			vm.defaultCheckedKeys = [];
			vm.deviceUrl = '';
			axios.post('${ctx}/system/deviceGroup/getFullDeviceGroupList.action',stringify(params)).then(function(response){
				let data = response.data
				vm.groupData = data.rows;
				if(vm.groupData.length>0){
					vm.defaultexpandedKeys.push(vm.groupData[0].id);
					vm.defaultCheckedKeys.push(vm.groupData[0].children[0].id);
					vm.params_device.group_id =vm.groupData[0].children[0].id;
					vm.rowDataGroup = vm.groupData[0].children[0];
					vm.$nextTick(function(){
						vm.$refs.groupTree.setCurrentKey(vm.params_device.group_id);
						vm.deviceUrl ="${ctx}/ups/queryUpsInfosListByGroupId.action"
					})
				}
			}).catch(function(error){})
		},
		queryDeviceGroupOption(){
			var vm = this;
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
				let data = response.data
				vm.deviceGroupOptions = data;
				if(vm.deviceGroupOptions.length>0){
					vm.deviceDialogForm.groupId = vm.deviceGroupOptions[0].id;
				}
			}).catch(function(error){});
		},
		// 添加设备
		addDevice(){
	    	var vm = this;
			vm.deviceDialogTitle = '<%=rb.getString("TianJia")%>';
			vm.showDeviceDialog = true;
			vm.viewDeviceDialog = false;
			vm.queryDeviceGroupOption();
		},
		// 新增设备提交
		addDeviceSubmit(){

			var vm = this,
				urls='${ctx}/ups/addDevice.action',
				snStr = vm.deviceDialogForm.serialnumber||'',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
				
			var params = {
					group_id:vm.deviceDialogForm.groupId,
					serialNumber: list.join(";"),
				};
			
			vm.$refs.deviceDialogForm.validate((valid) => {
				if(valid){
					axios.post(urls,stringify(params)).then(function(response){
						let data = response.data;
						if ( data.success ){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});	
							vm.$refs.ctableDevice.refresh();
							vm.closeDeviceDialog();
						}else {
							vm.$message.error(data.message)
						}
					}).catch(function(error){})
				}else{
					return false;
				}
		    })
		},
		// 关闭设备弹窗事件
		closeDeviceDialog(){
			var vm = this,
				params={
					serialNumber:'',
					groupId:'',
				};
			
			vm.$refs.deviceDialogForm.resetFields();
			Object.assign(vm.deviceDialogForm,params);
			vm.showDeviceDialog = false;
		},
		uploadDevice(){ // 确定导入
			var vm = this;
			if(vm.fileParams.FileName){
				vm.$refs.upload.submit();
			}else{
				vm.typeFlag = true;
				vm.selectFlag = true;
			}
			
		},
		checkFile(res,file){    //发送请求，校验device文件内容 
			var vm = this;
			if(res.success){
				if(res.suc_count>0){
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				}else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
				vm.showImportCard = false;
				vm.$refs.ctableDevice.refresh();
				vm.closeFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.upload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		/**
		*选择文件后，校验格式，并赋值页面显示 
		*@param file：文件名称
		*/
		fileChange(file,fileList){    
			var vm = this;
			vm.selectFlag = false;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xls' || file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' 
			vm.typeFlag = typeFlag;
			
			if(typeFlag){
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name
				vm.fileParams.group_id = vm.params_device.group_id;
			}else {
				vm.fileName = '';
			}
		},
		fileSelect(){  // 导入文件按钮
			var vm =this;
			vm.$refs.upload.clearFiles();
			vm.$refs['file_up'].click();
		},
		closeFileSelect(){ // 关闭文件选择
			var vm = this;
			vm.showImportBox = false;
			vm.fileName = '';
			vm.typeFlag = true;
			vm.selectFlag = false;
			vm.$refs.upload.clearFiles();
		},
		// 添加一级 设备组
		addDeviceGroup(){
			var vm = this;
			vm.showDeviceGroupDialog = true;
			vm.viewDeviceGroupDialog = false;
			vm.dialogTitle = '<%=rb.getString("TianJia")%>';
			vm.deviceGroupDialogWidth = '500px';
			vm.groupWindowType = 'add'
		},
		// 一级 设备组 新增/修改提交 
		addDeviceGroupSubmit(){
			var param={}  , vm = this , url;
				param.groupName = vm.groupForm.groupName;
				if (vm.groupWindowType == "add") {
					url = "${ctx}/system/deviceGroup/addTopDevice.action";
				} else {
					url = "${ctx}/system/deviceGroup/modTopDevice.action";
					param.groupId = vm.rowDataGroup.id;
				}
		    	vm.$refs.deviceGroupDialogForm.validate((valid) => {
		    		if(valid){
		    			axios.post(url,stringify(param)).then(function(response){
		    				let data = response.data;
		    				if ( data.success ){
		    					vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success',
								});
								vm.queryGroupList(vm.queryGroupSearchText);
		    				}else {
		    					vm.$message.error(data.message)
		    				}
		    				vm.closeDeviceGroupDialog();

		    			}).catch(function(error){})
		    		}else{
		    			return false;
		    		}
		    	})
		},
		// 关闭 一级 设备组弹窗
		closeDeviceGroupDialog(){
			var vm = this,
				params={
					groupName:'',
				};
			
			vm.showDeviceGroupDialog = false;
			vm.$refs.deviceGroupDialogForm.resetFields();
			Object.assign(vm.groupForm,params);
		},
		// 二级 设备组 新增/修改提交 
		addSubGroupSubmit(){
			var param={}, vm = this, url;
				param.name = vm.subGroupForm.groupName;
				param.desc = vm.subGroupForm.description;
				param.ids = vm.subGroupForm.devices;
				if (vm.subGroupWindowType == "add") {
					param.pid = vm.rowDataGroup.id,
					url = "${ctx}/ups/addDeviceGroup.action";
				} else {
					url = "${ctx}/ups/modDeviceGroup.action";
					param.group_id = vm.rowDataGroup.id;
				}
		    	vm.$refs.subGroupDialogForm.validate((valid) => {
		    		if(valid){
		    			axios.post(url,stringify(param)).then(function(response){
		    				let data = response.data;
		    				if ( data.success ){
		    					vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success',
								});	
								vm.$refs.ctableDevice.refresh();
								vm.queryGroupList(vm.queryGroupSearchText);
		    				}else {
		    					vm.$message.error(data.message)
		    				}
		    				vm.closeSubGroupDialog();

		    			}).catch(function(error){})
		    		}else{
		    			return false;
		    		}
		    	})
		},
		// 关闭 二级 设备组弹窗
		closeSubGroupDialog(){
			var vm = this,
				params={
					groupName:'',
					description:'',
					devices:''
				};
			
			vm.showSubGroupDialog = false;
			vm.$refs.subGroupDialogForm.resetFields();
			Object.assign(vm.subGroupForm,params);
			vm.deviceRightUrl = '';
		},
		/**
		* 设备选择变化时，更新选择设备记录
		* @param value:
		*/
		devicesChange(value) {// 
			var vm = this;
			vm.$nextTick(function(){
				var rows = vm.$refs.subGroupDevicePairgrid.getData();
				vm.subGroupForm.devices = rows.map(function(row){ return row.ups_code ;}).sort().join(',');
			})
		},
		querySubGroupDeviceList(val){
			var vm = this;
			vm.querySubGroupDeviceParams.searchText = val;
		},
		//点击页面其他地方菜单收起
		handerClose(){ 
	        this.$refs.menuGroup.hide();
	        this.$refs.menuDevices.hide();
	    },
		groupRowClick(data,node,ev){
			var vm = this;
			if(!data.children){
				vm.rowDataGroup = data;
				vm.params_device.group_id = data.id;
				vm.$nextTick(function(){
					vm.deviceUrl ="${ctx}/ups/queryUpsInfosListByGroupId.action"
					vm.$refs.ctableDevice.clearSelection();
				})
			
				
			}
		},
		/**
		 * 点击设备组操作 生成下拉选项
		 * @param row:当前点击项数据 
		 * 操作项： 1.信息  2.修改  3.删除  
		*/
		groupOpClick(node,data,ev){
			var vm = this,addShowFlag = false,editDisFlag=false;
			vm.rowDataGroup = data;
			if(data.children){
				addShowFlag = true
			}else{
				addShowFlag = false
			}
			if(data.built_in == "1" || (data.write != undefined && data.write != '1')){
				editDisFlag = true;
			}
	    	vm.menusGroup= [
				{label:'Add Subgroup',cls:"el-icon el-icon-operation-add CODE_UPS hidden",code:'add',show:addShowFlag},
		        {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info',show:!addShowFlag},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_UPS hidden",code:'edit',disable:editDisFlag},
		        {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_UPS hidden",code:'del',disable:editDisFlag}
		    ]
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.menuGroup.show(ev);
	    	});
			event.stopPropagation();
		},
		
		/**
		 * 设备组更多操作栏 单点方法 
		 * @param ev:当前点项
		*/
	    clickMenu(ev){ 
	    	var vm = this;
	    	var codes = {
				add:vm.addSubgroup,
	    		info:vm.viewSubGroupInfo,
	    		edit:vm.modifyGroup,
	    		del:vm.deleleGroup
	    	}
	    	if(codes[ev.code]){
	    		codes[ev.code](vm.rowDataGroup.id)
	    	}
	    },
		// 添加二级 设备组
		addSubgroup(){
			var vm = this;
			vm.showSubGroupDialog = true;
			vm.viewSubGroupDialog = false;
			vm.dialogTitle = '<%=rb.getString("TianJia")%>';
			vm.subGroupDialogWidth = '960px';
			vm.subGroupWindowType = 'add';
			vm.deviceRightUrl = '';
		},
		/**
		* 查看设备组详情
		* @param id 传入当前数据的id	
		*/
	    viewSubGroupInfo(id){
			var vm = this,
				params={
					groupName:vm.rowDataGroup.group_name,
					description:vm.rowDataGroup.description,
				};
			Object.assign(vm.subGroupForm,params);
			vm.showSubGroupDialog = true;
			vm.dialogTitle = '<%=rb.getString("XinXi")%>';
			vm.subGroupDialogWidth = '960px';
			vm.subGroupWindowType = 'info';
			vm.viewSubGroupDialog = true;
			vm.deviceRightUrl = '${ctx}/ups/querySelectUPS.action?group_id='+id;
	    },
		/**
		* 修改设备组
		* @param id 传入当前数据的id	
		*/
	    modifyGroup(id){
			var vm = this,
				params={
					groupName:vm.rowDataGroup.group_name,
					description:vm.rowDataGroup.description,
				};
			
			
			if(vm.rowDataGroup.children){
				vm.showDeviceGroupDialog = true;
				vm.viewDeviceGroupDialog = false;
				vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
				vm.deviceGroupDialogWidth = '500px';
				vm.groupWindowType = 'modify';
				Object.assign(vm.groupForm,params);
			}else{
				vm.showSubGroupDialog = true;
				vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
				vm.subGroupDialogWidth = '960px';
				vm.subGroupWindowType = 'edit';
				vm.viewSubGroupDialog = false;
				vm.deviceRightUrl = '${ctx}/ups/querySelectUPS.action?group_id='+id;
				Object.assign(vm.subGroupForm,params);
			}
			
	    },
		/**
		* 删除设备组
		* @param id 传入当前数据的id	
		*/
	    deleleGroup(id){
	    	var vm = this;
			if(vm.rowDataGroup.children){
				vm.delGroupType = 'stair';
			}else{
				vm.delGroupType = 'second';
			}
	    	this.showDeviceGroupInfo = true;
	    },
		 //删除设备组确认
		deleteDeviceGroup(){
			var vm = this,
				urls="",
				params = {};
			if(vm.delGroupType == 'stair'){
				params.groupId = vm.rowDataGroup.id
				urls = '${ctx}/system/deviceGroup/delTopDevice.action'
			}else{
				params.id = vm.rowDataGroup.id
				urls = "${ctx}/system/deviceGroup/deleteDeviceGroup.action"
			}
			axios.post(urls,stringify(params)).then(function(response){
				let data = response.data;
				if ( data.success ){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success'
					});	
					vm.showDeviceGroupInfo = false;
					vm.queryGroupList(vm.queryGroupSearchText);
				}else {
					vm.$message.error(data.message)
				}
				
			}).catch(function(error){})
		},
		// 输入框搜索
	    searchResult(val){
	    	this.params_device.searchText = val;
	    	this.$refs["ctableDevice"].refresh();	 
		},
		/**
		* 选择的批量数据
		* @param selection:传入批量数据对象
		*/
	    batchSelect(selection){
	    	var vm = this;
	    	vm.selection = selection;
	 	},
		// 打开导入弹窗
		importDevice(){
			this.showImportCard = true;
		},
		// 关闭导入弹窗
		closeImportDevice(){
			this.showImportCard = false;
		},
		exportDevice(){ // 导出
			var vm = this,
				params={},
				exportUrl ='${ctx}/ups/exportUpsToCSV.action';
			params.groupId = vm.params_device.group_id;
			params.searchText = vm.params_device.searchText;
			exportByForm(exportUrl,params);
		},
		/**
		 * 设备更多操作按钮
		 * @param row:添加下拉项
		*/
		optDeviceClick(row,ev){ // 操作项： 1.移动到设备组 2.删除 
	    	var vm = this,
	    		writable = vm.rowDataGroup.write == '1';
		
			vm.rowDataDevice = row;
			
	    	vm.menusDevices= [
				  {label:'<%=rb.getString("YiDongDaoSheBeiZu")%>',cls:"el-icon el-icon-moveGroup CODE_UPS hidden",code:'move',disable: !writable},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_UPS hidden",code:'deleteDevice',disable: !writable}
		    ];
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.menuDevices.show(ev);
	    	});
	    },
	    clickDeviceMenu(ev){ //单点击方法 -- 设备列表 
	    	var vm = this;
	    	var codes = {
				move:this.moveToGroup,
	    		deleteDevice:this.deleteDevice
	    	}
	    	if(codes[ev.code]){
				codes[ev.code](vm.rowDataDevice.ups_code)
	    	}
	    },
	
		/**
		* 当前数据移动设备到设备组
		* @prame code 当前数据
		*/
	    moveToGroup(code){
			var vm = this;
			vm.moveDevice.groupId = [] 
			vm.moveDevice.code = code
			vm.moveDevice.showMoveInfo = true;
			vm.moveDevice.dialogTitle = '<%=rb.getString("YiDongDaoSheBeiZu")%>';
			vm.moveDevice.windowWidth = '550px';
			vm.moveDevice.groupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?type=3&no_group_id=' + vm.params_device.group_id + "&rd="+Math.random().toString()
	    },
	    movecells(){ //批量操作 -- 移动设备到设备组
	    	var vm = this,
				idsList = [];
    	    
			if(vm.selection){
				vm.selection.map((item)=>{
					idsList.push(item.ups_code)	
				});
			} 
			vm.moveDevice.code = idsList.join(',');
			vm.moveDevice.groupId = [] 
			vm.moveDevice.showMoveInfo = true;
			vm.moveDevice.dialogTitle = '<%=rb.getString("YiDongDaoSheBeiZu")%>';
			vm.moveDevice.windowWidth = '550px';
			vm.moveDevice.groupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?type=3&no_group_id=' + vm.params_device.group_id + "&rd="+Math.random().toString()
			
		
	    },
		moveTrue(){ // 确定移动
			var vm = this,
				url = "${ctx}/ups/moveUPSToDeviceGroup.action",
				obj = {},
				str = '',
				code = vm.moveDevice.code;

			obj.toGroupId = vm.moveDevice.groupId
			obj.upsCodes = code;
			str = vm.moveDevice.groupId.toString()
			if(str !== ''){ // 如果没有选择设备组 提示先选择 只有在选择的时候才进行数据操作
				axios.post(url, stringify(obj)).then(function(response){
					let data = response.data;
					if(data.success){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							});	
							vm.moveDevice.showMoveInfo = false;
							vm.$refs.ctableDevice.refresh();
							vm.$refs.ctableDevice.clearSelection();
						}else{
							vm.$message({
								type:'error',
								message:data.message,
							})
						}
				})
			}else{
				vm.$message({
					type:'error',
					message:'<%=rb.getString("QingXuanZeSheBeiZu")%>',
				})
			}
		
		},
	
		/**
		 * 单独删除设备
		 * @param code：当前数据的 upsCodes
		*/
 		deleteDevice(code){
	    	var vm = this;
				url = "${ctx}/ups/delUpsInfo.action",
	    		params = {
					upsCodes: code
				};
	    	vm.$confirm('<%=rb.getString("ShanChuSheBeiHeShuJu")%>','<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning'
			}).then(()=>{
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});	
						vm.$refs.ctableDevice.refresh();
					}else {
						vm.$message.error(data.message)
					}
					
				}).catch(function(error){})
				
			}).catch(()=>{
				
			})
	    },
		// 批量删除
	    deleteCells(){
	    	var vm = this,
	    		params = {},
	    		url = "${ctx}/ups/delUpsInfo.action",
				idsList = [];
			vm.selection.map((item,index) => {
				idsList.push(item.ups_code)
			})
    	    params.upsCodes = idsList.join(',');
	    
	    	vm.$confirm('<%=rb.getString("ShanChuSheBeiHeShuJu")%>','<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning'
			}).then(()=>{
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						vm.$message({
					    	message: '<%=rb.getString("ChengGong")%>' ,
							type:'success',
						})
						vm.$refs.ctableDevice.refresh();
						vm.$refs.ctableDevice.clearSelection();
					}else {
						vm.$message.error(data.message)
					}
					
				}).catch(function(error){})
				
			}).catch(()=>{
				
			})
	    },
	    exportTemplate(){ // 导出模板
	    	
	    	var vm = this;
    	    var params = {};
    	    let url = '${ctx}/ups/downloadImportUpsTemplate.action';
		   
        	var bool = checkParams(params)
			if(!bool) return false;
        	exportByForm(url,params)
	    },
		isNumeric(str) {
			if(str.length==0){
				return false;
			}
			for(var i=0;i<str.length;i++){
				if(str.charAt(i)<"0" || str.charAt(i)>"9"){
					return false;
				}
			}
			return true;  
		},
	    
	},
	computed:{
		groupWritable() {
			var vm = this,
				write = true;
			
			if(vm.rowDataGroup && vm.rowDataGroup.write != '1') {
				write = false;
			}
			
			return write;
		},
		optWriteShow() {
			return writableMap['CODE_UPS'] == true;
		},
	},
	mounted(){
		var vm = this;
		vm.init();
	}
	
});

</script>