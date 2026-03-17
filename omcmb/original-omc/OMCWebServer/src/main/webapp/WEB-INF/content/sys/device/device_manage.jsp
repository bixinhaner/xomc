<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
#deviceMgmt{
	display:flex;
	border:1px solid #E9E9E9;
}
.panelDefault{
	width:99.8%;
	height:99.8%;
}
.groupMgmt {
	width:300px;
	height: 100%;
	border-right:1px solid #E9E9E9;
}
.deviceList {
	flex-grow:1;
	overflow:auto;
}
.splitTitle{
	font-size:14px;
	font-weight:bold;
	color:#363B4E;
	display:inline-block;
	height:34px;
	line-height:34px;
	padding-left:10px;
}
#importDeviceCard .el-card__body{
	padding:30px 30px 0;
	background:#FFF;
	border:none;
	height:186px;
}
#importDeviceCard .el-card__header{
	padding:0 20px;
	border-bottom:1px solid #E9E9E9;
	color:#333;
}
#importDeviceCard .el-card__footer{
	border:none;
	height:24px;
	line-height:24px;
	padding:0 30px 20px;
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
.importBox{
	display:inline-block;
	height:20px;
	width:20px;
	margin-top:4px;
	cursor:pointer;
}
#deviceMgmt .el-upload__tip{
	margin-top:2px;
}
#deviceMgmt .treeItemBoxCls{
	width: 300px;
	position: relative;
}
#deviceMgmt .treeItemBoxCls .operCls{
	position:absolute;
	right:3px;
	z-index: 66;
}
#deviceMgmt  .groupMgmt .el-input.el-input--small{
	width: 200px;
}
#deviceMgmt  .groupMgmt .el-tree-node__content{
	height: 30px;
}
#deviceMgmt .groupTreeBox{
	height: calc(100% - 44px);
	overflow: auto;
}
#deviceMgmt .ItemLabelCls{
	display: inline-block;
	width: 220px;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
#deviceMgmt .headQueryBox{
	display: flex;
	align-items: center;
}
.importWarp{
	width:600px;
	z-index:999;
	position:absolute;
	right:40px;
	top:50px;
}
.importWarp .el-icon-close{
	top:0;
}
.importWarp .el-card__header {
	border-bottom:1px solid #E4E7EC;
}
.importWarp .el-card__body {
	background: #fff;
	border: none !important;
	padding: 20px 38px;
}
.el-form-item{
	margin-bottom:26px !important;
}

.el-form-item__content{
	line-height:26px;
}
.seleceWidth .el-input { width: 260px; }
.tipSty {
	margin-top:10px;
}
.tipSty input {
	margin-right:8px;
}
</style>
<!-- 设备 -->
<div id="deviceMgmt" class="panelDefault" style="overflow:hidden;">
	<div class="groupMgmt" style="position:relative;display:flex;flex-direction:column;">
		<!-- 按钮   添加设备组 -->
		<div class="circleIcon hidden" :class="[enbFlag=='true'?'CODE_ENB_DEVICE_REGISTER':'CODE_CPE_DEVICE']" style="top:7px;" v-if="adminFlag">
			<span class="el-icon el-icon-circle-add" @click="addDeviceGroup"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<div class="splitTitle" style="padding:10px;"><%=rb.getString("SheBeiZu")%></div>
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
						<span class="el-icon el-icon-operation-more operCls" @click="groupOpClick(node,data,event)" v-clickoutside="handerClose"></span>
					</div>
				</el-tree>
			</div>
			<el-cmenu ref="menuGroup" :data="menusGroup" @click="clickMenu"></el-cmenu>
		</div>
	</div>
	<div class="deviceList">
		<!-- 按钮  -- 添加设备 -->
		<div class="circleIcon hidden" :class="[enbFlag=='true'?'CODE_ENB_DEVICE_REGISTER':'CODE_CPE_DEVICE']" style="right:110px;top:8px;">
			<span class="el-icon el-icon-circle-add" @click="addDevice"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		
		<!-- 按钮  -- 导入 -->
		<div class="circleIcon hidden" :class="[enbFlag=='true'?'CODE_ENB_DEVICE_REGISTER':'CODE_CPE_DEVICE']" style="right:55px;top:8px;">
			<span class="el-icon el-icon-circle-import" @click="importDevice"></span>
			<div class="titleButtonText"><%=rb.getString("DaoRu")%></div>
		</div>
		
		<!-- 按钮  -- 导出 -->
		<div class="circleIcon" style="top:8px;">
			<span class="el-icon el-icon-circle-export" @click="exportDevice"></span>
			<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
		</div>
		
		<el-ctable ref="ctableDevice" :url="deviceUrl" :height="height" :row-key="tableRowKey" :query-params="params_device" id="cpeOrEnbDeviceTable" 
			pagination="true" :rownumber=true @row-click="slectDeviceMethod" @selection-change='batchSelect'>
			
			<!-- 模糊查询 -->
			<template slot="toolbar">
				<div class="headQueryBox">
					<h3 style="padding-left: 10px;">{{deviceTitle}}</h3>
					<el-query v-if="enbFlag == 'true'" type="normal" @query="searchResult" placeholder="<%=rb.getString("XiaoZhanBianMa")%>"></el-query>
					<el-query v-if="enbFlag == 'false'" type="normal" @query="searchResult" placeholder="<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("MACDiZhi")%>"></el-query>
				</div>
			</template>
			
			<el-table-column v-if="batchShow && groupWritable" label='' width="50" type="selection" :reserve-selection="true" prop="ck"></el-table-column>
			<el-table-column v-if="batchShow" label='' width="30" prop="">
				<template slot-scope="scope">
            		<div class="el-icon el-icon-operation-more" @click="optDeviceClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
          		</template>
			</el-table-column>
			<el-table-column prop="connection_status" width="50">
				<template slot-scope="scope">
					<div :class="{
						'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
						'':scope.row.have_connected==2,
						'conn_exc':scope.row.connection_status=='Exception',
						'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
				</template>
			</el-table-column>
			<el-table-column v-if="enbFlag == 'true'" label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100" prop="serial_number"></el-table-column>
			<el-table-column v-if="enbFlag == 'true'" label='<%=rb.getString("HostName")%>' min-width="100" prop="host_name"></el-table-column>
			<el-table-column v-if="enbFlag == 'true'" label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="mac_address"></el-table-column>
			<el-table-column v-if="enbFlag == 'true'" label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="100" prop="group_name"></el-table-column>
			<el-table-column v-if="enbFlag == 'false'" label='<%=rb.getString("CPEXuLieHao")%>' min-width="100" prop="serial_number"></el-table-column>
			<el-table-column v-if="enbFlag == 'false'" label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="macaddress"></el-table-column>
			<el-table-column v-if="enbFlag == 'false'" label='IMSI' min-width="100" prop="imsi"></el-table-column>
			<el-table-column label='<%=rb.getString("JingDu")%>' min-width="100" prop="longitude"></el-table-column>
			<el-table-column label='<%=rb.getString("WeiDu")%>' min-width="100" prop="latitude"></el-table-column>
			<el-table-column label='<%=rb.getString("GaoDu")%>' min-width="100" prop="height"></el-table-column>
			<el-table-column v-if="enbFlag == 'false'" label='<%=rb.getString("JuLi")%>' min-width="100" prop="distance"></el-table-column>
		</el-ctable>
		<el-cmenu ref="menuDevices" :data="menusDevices" @click="clickDeviceMenu"></el-cmenu>
	</div>

	<el-dialog :title='dialogTitle' :visible.sync="showWindowInfo" ref="windowDialog" :width="windowWidth" 
		:close-on-click-modal="false"  :url="dialogUrl" @close='closeDialog'  @success="openDialogSuc">
			
	</el-dialog>
	<el-dialog title='<%=rb.getString("XiuGai")%>' :visible.sync="showModifyDevice" ref="modifyDialog" :width="windowWidth" 
		:close-on-click-modal="false"  @close='showModifyDevice = false' >
		<el-form ref="modifyForm" label-position="top" :model="deviceForm" :rules="deviceRules">
			<el-form-item v-show="enbFlag == 'true'">
				<%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("MaoHao")%> {{deviceName}}
			</el-form-item>
			<el-form-item v-show="enbFlag == 'false'">
				<%=rb.getString("MACDiZhi")%><%=rb.getString("MaoHao")%> {{deviceName}}
			</el-form-item>
			<el-form-item label="<%=rb.getString("JingDu")%>" prop="longitude">
				<el-input v-model="deviceForm.longitude"></el-input>
			</el-form-item>
			<el-form-item label="<%=rb.getString("WeiDu")%>" prop="latitude">
				<el-input v-model="deviceForm.latitude"></el-input>
			</el-form-item>
			<el-form-item label="<%=rb.getString("GaoDu")%>" prop="height">
				<el-input v-model="deviceForm.height"></el-input>
			</el-form-item>
			<el-form-item v-show="enbFlag == 'false'" label="<%=rb.getString("JuLi")%>" prop="distance">
				<el-input v-model="deviceForm.distance"></el-input>
			</el-form-item>
		</el-form>
		<div>
			<el-button type="primary" @click="saveModifyDevice"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="showModifyDevice = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!--添加新一级设备组-->
	<el-dialog 
		:title='dialogTitle' 
		:visible.sync="showDeviceGroupDialog" 
		ref="deviceGroupDialog" 
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
			@submit.native.prevent
		 >
			<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupName'>
				<el-input :disabled="viewDeviceGroupDialog" v-model="groupForm.groupName" style='width:200px;padding-top:7px;'></el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("MiaoShu")%>' prop='description' v-show="false">
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
	<!-- 批量操作 -->
	<el-bulk target="cpeOrEnbDeviceTable" :list="deviceSelectData" :row-key="tableRowKey" show-prop="select_name"
		:message="bulkTableMessage">
		<template slot="button">
			<a class="linkbutton" @click="deleteCells"><span><%=rb.getString("PiLiangShanChu")%></span></a>
			<a class="linkbutton" @click="movecells"><span><%=rb.getString("YiDongDaoSheBeiZu")%></span></a>
		</template>
	</el-bulk>
		
	<!-- 导入文件框 -->
	<div>
		<transition name='el-zoom-in-top'>
			<el-card v-show='showImportCard' id='importDeviceCard' class='importWarp'>
				<div slot='header'>
					<span><%=rb.getString("DaoRuSheBei")%></span>
					<span class='el-icon el-icon-close' style='float:right;font-size:16px;' @click='closeImportDevice'></span>
				</div>
				<div>
					<el-form label-position="top" ref="importDevicesForm" :model='importDevicesForm' :rules='importRules' label-width="140px"> 
						<el-form-item v-show="enbFlag == 'false'" prop='type' label="" style='margin-bottom:10px;'>
							<span style="font-size:14px;color:#333333;margin-right:20px;">Input Type</span>
							<el-radio-group v-model="importDevicesForm.type">
								<el-radio label="mac" style='margin-right:30px;'>MAC</el-radio>
								<el-radio label="sn" style='margin-bottom:0px;'><%=rb.getString("CPEBianMa")%></el-radio>
							</el-radio-group>
						</el-form-item>
						<el-form-item label="<%=rb.getString("WenJian")%>"  prop="fileName">
		                  	<el-upload 
		                  		:before-upload='beforeUpload' 
		                  		:on-success='checkFile' 
		                  		:on-change="fileChange"  
		                  		:show-file-list=false ref="upload"
							    :action="importDevicesForm.uploadFileUrl" 
							    :data="fileParams" name="uploadFile" 
							    :auto-upload="false"
							    accept=".xlsx,.csv">
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:260px;">
									<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
								</el-input>								
								<a slot="trigger" ref="file_up"></a>
							</el-upload>
							<div style='color:#999;line-height:24px;'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span><%=rb.getString("DaoRuWenJianTiShi")%>.
								<span style="cursor:pointer;" @click="exportTemplate">
									<span style='vertical-align:top' class='el-icon el-icon-common-download'></span>
									<span v-show="enbFlag == 'true'" style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
									<span v-show="enbFlag == 'false' && importDevicesForm.type == 'mac'" style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%>(MAC)</span>
									<span v-show="enbFlag == 'false' && importDevicesForm.type == 'sn'" style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%>(<%=rb.getString("CPEBianMa")%>)</span>
								</span>
							</div>	
	                    </el-form-item> 
						<el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' v-show="enbFlag == 'true'">
							<el-select v-model="importDevicesForm.group_id" class='seleceWidth'>
								<el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
							</el-select>
						</el-form-item>
					</el-form>
				</div>
				<div slot='footer'>
					<el-button-group size="mini">
		    			<el-button type="primary" size="mini" @click="uploadDevice"><%=rb.getString("QueDing")%></el-button>
		    			<el-button size="mini" @click="closeImportDevice"><%=rb.getString("QuXiao")%></el-button>
		    		</el-button-group>
				</div>
				
			</el-card>
		</transition>
	</div>
	  <!--上传文件的用的表单 -->
	<form enctype="multipart/form-data" method="post" id="importDeviceForm">
	    <input name="fileSize"  value="" hidden="true">
	    <input name="operType" value="" hidden="true">
	    <input name="uploadFile"  id="importDeviceFile"  type="file" style="display: none;">
	</form>
	<!-- 下载模板用的表单 -->
	<form id="exportDeviceForm" style="display:none" method="post"></form>
	
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
</div>


<script>
var deviceVue = new Vue({
	el:'#deviceMgmt',
	data(){
		var vm = this;
		/**
		* 验证输入的字符长度
		* @param str{string}   value值
		* @param num{number}   长度
		*/ 
		var lessThenLength = function(str,num){
				str += '';
				var index = str.indexOf('.');
				if(index > -1){
					return str.substring(index+1).length <= num
				}
				return true;
			},
			// 经度 表单验证规则
			longitudeValidator = function(rule,value,cb) {
				if(value) {
					if(!isNaN(value) && Math.abs(value)<=180 && lessThenLength(Math.abs(value),6)) {
						cb();
					}else {
						cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
					}
				}else {
					cb();
				}
			},
			// 维度 表单验证规则
			latitudeValidator = function(rule,value,cb) {
				if(value) {
					if(!isNaN(value) && Math.abs(value)<=90 && lessThenLength(Math.abs(value),6)) {
						cb();
					}else {
						cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
					}
				}else {
					cb();
				}
			},
			// 高度 表单验证规则
			heightValidator = function(rule,value,cb) {
				if(value) {
					if(value<=99999999 && value-0>=0) {
						cb();
					}else {
						cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
					}
				}else {
					cb();
				}
			},
			// 距离 表单验证规则
			distanceValidator = function(rule,value,cb) {
				if(value) {
					if(value<=99999999 && value-0>=0) {
						cb();
					}else {
						cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
					}
				}else {
					cb();
				}
			},
			validateGroupName = (rule,value,callback) => {
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
			validateFileName = function(rule,value,callback) {
            	value = vm.fileName;     		
				if( value === '' || value === null || value === undefined) {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}else {
					callback();
				}
			};

		return {
			idsStr:'',			
			groupId:'',
			tableRowKey:'', // 表格row-key
			height:'100%',
			menusGroup:[],
			deviceUrl:'',
			serialNumber:'',
			searchText:'',
			groupData:[
				
			],
			enbFlag : "${ deviceType == 'eNB' }",
			params_device:{
				timeZone:timeZone,
				group_id:'',
				search_text:'',
				like_fields: 'serial_number,macaddress'
			},
			showWindowInfo:false,
			dialogTitle:'',
			dialogUrl:'',
			menusDevices:[],
			rowDataGroup:[],
			rowDataDevice:[],
			
			windowWidth:'500px',
			groupWindowType:'',
			slideUrl:'',
			slideTitle:'Add eNB',
			deviceTitle:'eNB',
			buttomOptShow: false,
			selection:'',
			showImportCard:false,
			selectIds:'',
			adminFlag:"${is_super_adm}" == "1" || "${operator_built_role}" == "1",
			showModifyDevice: false,
			deviceForm: {
				cell_code: '',
				longitude: '',
				latitude: '',
				height: '',
				distance: '',
				operator_code: operator_code
			},
			deviceRules: {
				longitude: [
					{validator: longitudeValidator}
				],
				latitude: [
					{validator: latitudeValidator}
				],
				height: [
					{validator: heightValidator}
				],
				distance: [
					{validator: distanceValidator}
				]
			},
			sasEnable: true,
			deviceName: '',
			deviceLinkOptions: [
				{text:'NLOS',value:'nlos'},
				{text:'PLOS',value:'plos'},
				{text:'LOS',value:'los'}
			],
			batchShow: '${ deviceType }'=='eNB'?writableMap['CODE_ENB_DEVICE_REGISTER']:writableMap['CODE_CPE_DEVICE'],
			showDeviceGroupInfo:false,
			deviceSelectData:[],
			selectDeviceList:[],
			bulkTableMessage:{
				title:'<%=rb.getString("YiXuanSheBei")%>',
				subTitle:'<%=rb.getString("MACDiZhi")%>',
				clear:'<%=rb.getString("QingKong")%>',
				cancel:'<%=rb.getString("QuXiao")%>'
			},
			stairGroupType:'',
			showDeviceGroupDialog:false,
			deviceGroupDialogHeight:'600px',
			deviceGroupDialogWidth:'500px',
			viewDeviceGroupDialog:false,
			groupForm:{
				groupName:'',
			},
			groupRules:{
				groupName:[
					{validator:validateGroupName,trigger:'blur'}
				],
			},
			delGroupType:'',
			queryGroupSearchText:'',
			defaultexpandedKeys:[],
			defaultCheckedKeys:[],
			//import
			deviceGroupOptions:[],
			importDevicesForm: {
				type:'mac',
                uploadFileUrl: '',
                group_id: ''
           	},
            fileParams:{}, 
            fileName:'',
			showFileTip:false,
			fileList:[],
			filePath:'',
			rowData:[],
			importRules: {
				fileName:[ {validator: validateFileName}]
            },
		}
	},
	methods:{
		deviceGroupChange(val){
			var vm = this;
			vm.importDevicesForm.group_id = val;
		},
		
		//导入
		importDevice(){
			var vm = this;
			if(vm.enbFlag == 'true'){
				axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
					var data = response.data;
					vm.deviceGroupOptions = data;
					vm.importDevicesForm.group_id = data[0].id;
				}).catch(function(error){})
			}
			
			vm.showImportCard = true;
		},
		/**
		* 文件上传成功函数 
		* @param res{object}   返回信息
		* @param file{object}  文件信息
		*/
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
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		fileChange(file,fileList){ 
			var vm = this;
			vm.fileName = file.name;
			vm.fileParams.FileName = file.name;
		},
		
		// 选择文件
		fileSelect(){  
			var vm =this;
			vm.$refs.upload.clearFiles();
			vm.$refs['file_up'].click();
		},
		
		// 移除导入文件
		closeFileSelect(){
			var vm = this;
			vm.fileName = '';			
			vm.$refs.upload.clearFiles();
		},
					
		/**
		* 文件上传之前
		* @param file{object}   文件信息
		*/ 
		beforeUpload(file){
			var vm = this, url = '',
				fileName = file.name,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
			if(vm.enbFlag == 'true'){
				url = '${ctx}/system/deviceGroup/uploadFile.action';
				fd.append('group_id',vm.importDevicesForm.group_id);
			}else{
				if(vm.importDevicesForm.type == 'mac'){
					url = '${ctx}/cell/CPE/uploadFile.action?importType=append';
				}else{
					url = '${ctx}/cell/CPE/uploadFile2.action?importType=append';
				}
				fd.append('group_id',vm.params_device.group_id);
			}
			fd.append('uploadFile',file); //文件流
			fd.append('FileName',fileName);//文件名

			axios.post(url,fd,config).then(function(res){
				if(res.data["success"]){	
					vm.$message.success('<%=rb.getString("ChengGong")%>');
					vm.$refs.ctableDevice.refresh();
					
					vm.showImportCard = false;
					vm.fileList = [];
					vm.fileName = '';
					vm.$refs.importDevicesForm.resetFields();						
				}else{
					vm.$message.error(res.data["msg"])
				}
			})
			
			return false;
		},
		
		/*确定导入*/
        uploadDevice() {
			var vm = this;
			vm.$refs.importDevicesForm.validate((valid) => {
                if (valid) {
                	vm.$refs.upload.submit();                   	
                }
            }) 				
		},
		// 关闭导入弹出框
		closeImportDevice(){
			var vm = this;
			vm.showImportCard = false;
			vm.fileList = [];
			vm.fileName = '';
			vm.$refs.importDevicesForm.resetFields();
		},
		// 初始化 更改设备类型显示
		init(){
			var vm = this;
			if ( vm.enbFlag == 'false') {
				vm.deviceTitle = 'CPE';
				vm.tableRowKey = 'macaddress';
				vm.serialNumber = '<%=rb.getString("CPEBianMa")%>/<%=rb.getString("MACDiZhi")%>';
				vm.bulkTableMessage = {title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("MACDiZhi")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'};
			}else{
				vm.tableRowKey = 'serial_number';
				vm.serialNumber = '<%=rb.getString("XiaoZhanBianMa")%>';
				vm.bulkTableMessage = {title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'};
			}
			vm.queryGroupList(vm.queryGroupSearchText);
		},
		queryGroupList(val){
			var vm =this,
				params={
					search_text:val,
					type: vm.enbFlag == 'true'?0:2
				};
			if(vm.enbFlag == 'true'){
				params.isEnb = 1;
			}else {
				params.isCpe = 1;
			}
			vm.queryGroupSearchText = val;
			vm.defaultexpandedKeys =[];
			vm.defaultCheckedKeys = [];
			vm.deviceUrl = '';
			axios.post('${ctx}/system/deviceGroup/getFullDeviceGroupList.action',stringify(params)).then(function(response){
				let data = response.data.rows;
				vm.groupData = data;
				
				if(vm.groupData.length>0){
					vm.defaultexpandedKeys.push(vm.groupData[0].id);
					vm.defaultCheckedKeys.push(vm.groupData[0].children[0].id);
					vm.params_device.group_id =vm.groupData[0].children[0].id;
					vm.rowDataGroup = vm.groupData[0].children[0];
					if(vm.enbFlag == 'true'){
						vm.$nextTick(function(){
							vm.$refs.groupTree.setCurrentKey(vm.params_device.group_id);
							vm.deviceUrl = '${ctx}/cell/cpeinfos/getEnbList.action';
						})
					}else{
						vm.$nextTick(function(){
							vm.deviceUrl = '${ctx}/system/device/cpe/queryCPEInfoPageList.action';
						})
					}
				}
			}).catch(function(error){})
		},
		// 新增设备组页面
		addDeviceGroup(){
			var vm = this;
			vm.showDeviceGroupDialog = true;
			vm.viewDeviceGroupDialog = false;
			vm.dialogTitle = '<%=rb.getString("TianJia")%>';
			vm.deviceGroupDialogWidth = '500px';
			vm.stairGroupType = 'add'
			
		},
		// 一级 设备组 新增/修改提交 
		addDeviceGroupSubmit(){
			var param={}  , vm = this , url;
				param.groupName = vm.groupForm.groupName;
				if (vm.stairGroupType == "add") {
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
			vm.$refs.deviceGroupDialogForm.resetFields();
			Object.assign(vm.groupForm,params);
			vm.showDeviceGroupDialog = false;
		},
		// 弹窗打开成功传递参数  
		openDialogSuc(){
			var vm = this;
			//vm.groupWindowType == 'singleDeviceGroup' 单个移动设备组
			//vm.groupWindowType == 'batchDeviceGroup'  批量移动设备组
			//vm.groupWindowType == 'add'    设备组-新建
			//vm.groupWindowType == 'modify' 设备组-修改
			//vm.groupWindowType == 'info'   设备组-详情			
			//vm.groupWindowType == 'addCPEOrENBDevice' 添加CPE or 添加ENB基站			
			//ENB:vm.selectIds = row.small_cell_code + '_' + row.product;  
			//CPE:vm.selectIds = row.cpe_code;			
			//vm.rowDataGroup.id: 左侧Device Group 的id
			//vm.enbFlag:标识 CPE-false or eNB-true		
			//vm.groupWindowType :当前操作的标识：add-新建设备组，modify-修改设备组，info-设备组详情，singleDeviceGroup-单个移动设备组，batchDeviceGroup-批量移动设备组，addCPEOrENBDevice-添加CPE or 添加ENB基站
			if(vm.groupWindowType == 'singleDeviceGroup'){
				eventBus.$emit('open-dialog',vm.selectIds,vm.params_device.group_id,vm.enbFlag);
			}else if(vm.groupWindowType == 'batchDeviceGroup'){
				eventBus.$emit('open-dialog-moveto',vm.idsStr,vm.params_device.group_id,vm.enbFlag);
			}else if(vm.groupWindowType == 'add' || vm.groupWindowType == 'modify' || vm.groupWindowType == 'info'){
				eventBus.$emit('addModifyInfo-dialog',vm.groupWindowType,vm.rowDataGroup.id,vm.enbFlag);
			}else {//else if(vm.groupWindowType == 'addCPEOrENBDevice'){}
				eventBus.$emit('addDevice-dialog',vm.rowDataGroup.id);
			}
		},
		// 关闭弹窗
		closeDialog(){
			var vm = this;
			vm.$refs.ctableDevice.refresh();
			vm.$refs.ctableDevice.clearSelection();
			vm.showWindowInfo = false;
			vm.dialogUrl = '';
		},
		//点击页面其他地方菜单收起
		handerClose(){ 
	        this.$refs.menuGroup.hide();
	        this.$refs.menuDevices.hide();
	    },
		// 二级设备组 行点击事件
		groupRowClick(data,node,ev){
			var vm = this;
			if(!data.children){
				vm.rowDataGroup = data;
				if ( vm.enbFlag == 'true' ){
					vm.params_device.group_id = data.id;
					vm.$nextTick(function(){
						vm.deviceUrl = '${ctx}/cell/cpeinfos/getEnbList.action';
						vm.$refs.ctableDevice.clearSelection();
					})
				} else {
					vm.params_device.group_id = data.id;
					vm.$nextTick(function(){
						vm.deviceUrl = '${ctx}/system/device/cpe/queryCPEInfoPageList.action';
						vm.$refs.ctableDevice.clearSelection();
					})
				}
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
			if(data.built_in == '1' || (data.write != undefined && data.write != '1')){
				editDisFlag = true;
			}
			var cls = {
					'true': ' CODE_ENB_DEVICE_REGISTER hidden',
					'false': ' CODE_CPE_DEVICE hidden'
				},
				mcls = cls[vm.enbFlag] || '';
	    	vm.menusGroup= [
				{label:'Add Subgroup',cls:"el-icon el-icon-operation-add"+mcls,code:'add',show:addShowFlag},
		        {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info',show:!addShowFlag},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit"+mcls,code:'edit',disable:editDisFlag},
		        {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete"+mcls,code:'del',disable:editDisFlag}
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
			vm.groupWindowType = 'add';
			vm.dialogTitle = '<%=rb.getString("TianJia")%>';
			vm.windowWidth = '1000px';
			vm.dialogUrl = '${ctx}/system/deviceGroup/toDiviceGroupPage.action';
			vm.showWindowInfo = true;
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
			Object.assign(vm.groupForm,params);
			vm.groupWindowType = 'info';			
			vm.dialogTitle = '<%=rb.getString("XinXi")%>';
			vm.windowWidth = '1000px';
			vm.dialogUrl = '${ctx}/system/deviceGroup/toDiviceGroupPage.action';
			vm.showWindowInfo = true;
	    },
		/**
		* 修改设备组
		* @param id 传入当前数据的id	
		*/
	    modifyGroup(id){
			var vm = this,
				params={
					groupName:vm.rowDataGroup.group_name,
				};
			if(vm.rowDataGroup.children){
				vm.showDeviceGroupDialog = true;
				vm.viewDeviceGroupDialog = false;
				vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
				vm.deviceGroupDialogWidth = '500px';
				vm.stairGroupType = 'modify';
				Object.assign(vm.groupForm,params);
			}else{
				vm.groupWindowType = 'modify';		
				vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
				vm.windowWidth = '1000px';
				vm.dialogUrl = '${ctx}/system/deviceGroup/toDiviceGroupPage.action';
				vm.showWindowInfo = true;
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
				params.groupId = vm.rowDataGroup.id;
				urls = '${ctx}/system/deviceGroup/delTopDevice.action'
			}else{
				params.id = vm.rowDataGroup.id;
				urls = "${ctx}/system/deviceGroup/deleteDeviceGroup.action"
			}
			axios.post(urls,stringify(params)).then(function(response){
				let data = response.data;
				if ( data.success ){
					vm.$message.success(data.message);
					vm.showDeviceGroupInfo = false;
					vm.queryGroupList(vm.queryGroupSearchText);
				}else {
					vm.$message.error(data.message)
				}
				
			}).catch(function(error){})
		},
		// 基站设备 搜索事件
	    searchResult(val){
	    	var vm = this;
	    	vm.params_device.search_text = val;
			setTimeout(()=>{
				vm.$refs.ctableDevice.refresh();
			},260)
	    },
		/**
		* 基站列表选中
		* @param selection{Array}   选中数据
		*/
	    batchSelect(selection){
	    	var vm = this;
			
	    	vm.selectDeviceList = selection;
			vm.deviceSelectData = selection.map((item)=>{
				//eNB
				if ( vm.enbFlag == 'true') {
					return Object.assign(item,{select_name:item.serial_number})
				}else{
					return Object.assign(item,{select_name:item.macaddress})					
				}
				
			});
	    	
	    },
		/**
		* 基站列表单行点击事件 生成 selectIds
		* @param row{Array}   行数据
		*/
	    slectDeviceMethod(row){
	    	if(this.enbFlag == "true"){
	    		this.selectIds = row.small_cell_code + '_' + row.product;
	    	}else{
	    		this.selectIds = row.cpe_code;
	    	}
	    },
		// 新增基站设备页面
		addDevice(){
	    	var vm = this;
	    	vm.groupWindowType = 'addCPEOrENBDevice';			
			vm.windowWidth = '520px';
			// 区分CPE,ENB，新建设备
		    if( vm.enbFlag == 'true' ){
		    	vm.dialogUrl = "${ctx}/system/deviceGroup/toAddDevice.action?device_type=eNodeB",
		        vm.dialogTitle = '<%=rb.getString("TianJiaJiZhan")%>';
		    } else {
		    	vm.dialogUrl = "${ctx}/system/deviceGroup/toAddDevice.action?device_type=CPE",
		        vm.dialogTitle = '<%=rb.getString("TianJiaCPE")%>';
		    }
		    vm.showWindowInfo = true;
		},
		
		// 导出设备列表
		exportDevice(){
			var vm = this,
				params={},
				exportUrl ='';
			
			if ( vm.enbFlag == 'true'){
				exportUrl = '${ctx}/system/device/enodeb/exportENBCsvFile.action';
			} else {
				exportUrl = '${ctx}/system/device/cpe/exportCPECsvFile.action';
			}
			params.timeZone = timeZone;
			params.group_id = vm.params_device.group_id;
			params.search_text = vm.searchText;
			params.like_fields = "serial_number";
			exportByForm(exportUrl,params);
		},
		/**
		* 基站设备列表 点击更多操作出现菜单
		* @param row{object}   行数据
		* @param ev{object}   event数据
		*/ 
		optDeviceClick(row,ev){ // 操作项： 1.移动到设备组 2.删除 
	    	var vm = this ; 
				vm.rowDataDevice = row,
				writable = vm.rowDataGroup.write == '1';
				
	    	if ( vm.enbFlag == 'true'){
				vm.menusDevices= [
		          {label:'<%=rb.getString("YiDongDaoSheBeiZu")%>',cls:"el-icon el-icon-moveGroup CODE_ENB_DEVICE_REGISTER  hidden",code:'move',disable: !writable},
		          {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_ENB_DEVICE_REGISTER  hidden",code:'modifyDevice',disable: !writable},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ENB_DEVICE_REGISTER  hidden",code:'deleteDevice',disable: !writable}
		    	]
			} else {
				vm.menusDevices= [
		          {label:'<%=rb.getString("YiDongDaoSheBeiZu")%>',cls:"el-icon el-icon-moveGroup  CODE_CPE_DEVICE hidden",code:'move',disable: !writable},
		          {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit  CODE_CPE_DEVICE hidden",code:'modifyDevice',disable: !writable},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete  CODE_CPE_DEVICE hidden",code:'deleteDevice',disable: !writable}
		    	]
			}
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.menuDevices.show(ev);
	    	});
	    },
		/**
		* 基站设备列表 菜单点击事件
		* @param ev{object}   行数据
		*/ 
	    clickDeviceMenu(ev){ //单点击方法 -- 设备列表 
	    	var vm = this;
	    	var codes = {
	    		move:this.moveToGroup,
				modifyDevice: this.modifyDevice,
	    		deleteDevice:this.deleteDevice
	    	}
	    	if(codes[ev.code]){
	    		if(vm.enbFlag == 'true'){
	    			codes[ev.code](vm.rowDataDevice)
	    		}else{
	    			codes[ev.code](vm.rowDataDevice.cpe_code)
	    		}
	    	}
	    },
		/**
		* 移动到设备组
		* @param code{object}   设备数据
		*/ 
	    moveToGroup(code){
			var vm = this;
			vm.groupWindowType = 'singleDeviceGroup';
			vm.dialogTitle = '<%=rb.getString("YiDongDaoSheBeiZu")%>';
			vm.windowWidth = '550px';
			vm.dialogUrl = '${ctx}/system/deviceGroup/toDiviceGroupPage.action?code=move';
			//vm.groupWindowType = vm.rowDataDevice.product;
			vm.showWindowInfo = true;
	    },
		// 移动到设备组 批量操作
	    movecells(){
	    	var vm = this;
	    	let params = {}
	    	if(vm.enbFlag == 'true'){
	    		var idsStr = ""
    	    	vm.selectDeviceList.map((item,index) => {
    	    		idsStr += item.small_cell_code + '_' + item.product + ','
    	    		return idsStr;
    	    	})
    	    	params.ids = idsStr
	    	}else{
	    		var idsStr = ""
    	    	vm.selectDeviceList.map((item,index) => {
    	    		idsStr += item.cpe_code+","
    	    		return idsStr;
    	    	})
    	    	params.cpeCodes = idsStr;
	    	}
			vm.idsStr = idsStr;
			vm.groupWindowType = 'batchDeviceGroup';
			vm.dialogTitle = '<%=rb.getString("YiDongDaoSheBeiZu")%>';
			vm.windowWidth = '550px';
			vm.dialogUrl = '${ctx}/system/deviceGroup/toDiviceGroupPage.action?code=move';
			//vm.groupWindowType = vm.rowDataDevice.product;
			vm.showWindowInfo = true;
	    },
		/**
		* 基站设备修改页面
		* @param code{object}   设备数据
		*/ 
		modifyDevice(code) {
			var vm = this,
				data = vm.$refs.ctableDevice.getData()||[],
				row = '',
				cell_code = '';
			
			vm.showModifyDevice = true;
			if(vm.enbFlag == 'true'){ // eNB device
	    		row = code;
				cell_code = row.small_cell_code;
				vm.deviceName = row.serial_number;
	    	}else{ // CPE device
	    		row = data.filter(function(item){
					return item.cpe_code == code;
				})[0];
				cell_code = row.cpe_code;
				vm.deviceName = row.macaddress;
	    	}
			
			if(row) {
				Object.assign(vm.deviceForm,{
					cell_code: cell_code,
					longitude: row.longitude,
					latitude: row.latitude,
					height: row.height,
					distance: row.distance,
				})
			}else {
				Object.assign(vm.deviceForm,{
					cell_code: '',
					longitude: '',
					latitude: '',
					height: '',
					distance: '',
				})
			}
		},
		// 基站设备修改确定
		saveModifyDevice(){
			var vm = this,
				url = '${ctx}/cell/topo/setLocationInfo.action';

			vm.$refs.modifyForm.validate(function(r){
				if(r) {
					axios.post(url, stringify(vm.deviceForm)).then(function(res){
						if(res.data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.showModifyDevice = false;
							vm.$refs.ctableDevice.refresh();
						}else {
							vm.$message({
								message: res.data.message,
								type:'error',
							});
						}
					});
				}
			})
		},
		/**
		* 基站设备删除
		* @param code{object}   设备数据
		*/ 
 		deleteDevice(code){
	    	var vm = this , msg='';
	    	let params = {};
	    	var url = ""
	    	if(this.enbFlag == 'true'){
	    		url = "${ctx}/system/deviceGroup/delCellinfo.action"
	    		params.ids = code.small_cell_code + '_' + code.product;
	    		msg='<div><%=rb.getString("QueDingShanChuSheBei")%></div> <div class="tipSty"><input id="delFlag" name="delFlag" type="checkbox" /><%=rb.getString("TongShiShanChuSheBeiShuJu")%></div>'
	    	}else{
	    		url = "${ctx}/cell/CPE/delCpeinfo.action"
	    		params.cpeCodes = code;
	    		msg = '<%=rb.getString("ShanChuSheBeiHeShuJu")%>'
	    	}
	    	vm.$confirm(msg,
	    		'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				dangerouslyUseHTMLString:true
			}).then(()=>{
				if(vm.enbFlag == 'true'){
					var enable = $("#delFlag").is(':checked');
					$("#delFlag").prop('checked',false);
					params.deleteOtherData = enable;
				}
				
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						if(this.enbFlag == 'true'){
							vm.$message.success(data.message)
						}else{
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							})
						}
						vm.$refs.ctableDevice.refresh();
					}else {
						vm.$message.error(data.message)
					}
					vm.deviceSelectData = [];
					vm.$refs.ctableDevice.clearSelection();
				}).catch(function(error){}) 
				
			}).catch(()=>{
				
			})
	    },
		// 基站设备 批量删除
	    deleteCells(){
	    	var vm = this,msg='';
	    	var url = "";
	    	let params = {}
	    	if(vm.enbFlag == 'true'){
	    		url = "${ctx}/system/deviceGroup/delCellinfo.action"
	    		var idsStr = ""
    	    	vm.selectDeviceList.map((item,index) => {
    	    		idsStr += item.small_cell_code + '_' + item.product + ','
    	    		return idsStr;
    	    	})
    	    	params.ids = idsStr;
	    		msg='<div><%=rb.getString("QueDingShanChuSheBei")%></div> <div class="tipSty"><input id="delFlag" name="delFlag" type="checkbox" /><%=rb.getString("TongShiShanChuSheBeiShuJu")%></div>'
	    	}else{
	    		url = "${ctx}/cell/CPE/delCpeinfo.action"
	    		var idsStr = ""
    	    	vm.selectDeviceList.map((item,index) => {
    	    		idsStr += item.cpe_code+","
    	    		return idsStr;
    	    	})
    	    	params.cpeCodes = idsStr;
	    		msg='<%=rb.getString("ShanChuSheBeiHeShuJu")%>';
	    	}
	    	
	    	vm.$confirm(msg,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancalButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				dangerouslyUseHTMLString:true
			}).then(()=>{
				if(vm.enbFlag == 'true'){
					var enable = $("#delFlag").is(':checked');
					$("#delFlag").prop('checked',false);
					params.deleteOtherData = enable;
				}
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data;
					if ( data.success ){
						//vm.$message.success(data.message)
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						})
						vm.$refs.ctableDevice.refresh();
					}else {
						vm.$message.error(data.message)
					}
					vm.deviceSelectData = [];
					vm.$refs.ctableDevice.clearSelection();
				}).catch(function(error){})
				
			}).catch(()=>{
				
			})
	    },
		// 关闭弹窗 刷新数据
	    refreshTtable(){
	    	this.showWindowInfo = false;
	    	this.$refs.ctableDevice.refresh()
	    },
		// 导出设备模板
	    exportTemplate(){
	    	
	    	//判断是eNB 还是 cpe的设备模板导出
	    	var vm = this;
    	    var params = {};
    	    let url = '';
		    if(vm.enbFlag == 'true'){
		    	url = "${ctx}/system/deviceGroup/downloadImportCellTemplate.action";
		    }else{
				params.type = vm.importDevicesForm.type;
		    	url = "${ctx}/cell/CPE/downloadImportCpeTemplate.action";
		    }
		    params.group_id = this.groupId;
	    	params.search_text = this.searchText;
    	    params.like_fields = "serial_number";
        	var bool = checkParams(params)
			if(!bool) return false;
        	exportByForm(url,params)
	    }
	},
	computed:{
		groupWritable() {
			var vm = this,
				write = true;
			
			if(vm.rowDataGroup && vm.rowDataGroup.write != '1') {
				write = false
			}

			return write;
		}
	},
	mounted(){
		var vm = this;
		this.init();
		eventBus.$on('close-dialog',this.closeDialog);
		eventBus.$off("add-enode").$on("add-enode",this.refreshTtable)
		eventBus.$off("query-group").$on("query-group",this.queryGroupList)
		axios.post('${ctx}/cell/topo/getSasEnableStatus.action',stringify({operator_code: operator_code})).then(function(res){
			if(res && res.data) {
				vm.sasEnable = !res.data.success
			}
		});
	}
	
});

</script>