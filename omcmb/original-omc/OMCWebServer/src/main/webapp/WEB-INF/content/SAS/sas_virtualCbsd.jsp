<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
	.el-notification__group .el-icon-close{
		position: absolute !important
	}
	.el-icon-close{
		
	}
	.selected-devices-info .el-icon-close{
		font-size: 16px !important;
		position: unset !important;
	}
	.el-card__body{
		border:none !important;
	}
	.el-message__closeBtn{
		position: absolute !important;
		right:15px !important
	}
	.slide-content{
		padding:0px
	}
	.slide-content{
		padding:0px !important
	}
	.queryInfo{
		display:inline-block;
		vertical-align:top;
	}
	.queryInfo label{
		display:block;
		margin-bottom:5px;
		line-height:26px;
	}

	#omcLogs .el-tabs__nav-wrap::after {
		right:120px;
	}
	.bottom-slider {
		position: absolute;
		bottom: -110px;
		width: calc(100% - 30px);
		padding: 12px 15px;
		background: #fff;
		box-shadow: 10px 0 50px rgba(0,0,0,0.15);
		transition: bottom .5s ease;
		z-index: 98;
	}
	.bottom-slider.show {
		bottom: 0px;
	}

	.selected-devices-info {
		padding: 15px 15px 15px 0;
		height: 400px;
		width: 400px;
		position: absolute;
		bottom: -450px;
		background: #fff;
		box-shadow: 10px 0 50px rgba(158,200,222,0.45);
		transition: bottom .5s ease;
		z-index: 90;
	}
	.selected-devices-info.show {
		bottom: 66px;
	}
	.enb-list-ctn {
		position:absolute;
		width: 100%;
		height: 100%;
		z-index: 100;
		top: 0;
		background-color: #fff;
		display: flex;
		flex-direction: column;
	}
	.list-title {
		font-size: 16px;
		color: #5A7B92;
		padding: 10px 0 10px 25px;
		border-bottom:1px solid #E4E7EC;
	}
	.list-title span{
		float: right;
		padding-right: 15px;
	}
	.list-body-box{
		width: calc(100% - 44px);
		padding-left: 25px;
	}
	.list-body-box .operation_deletes {
		float: right;
		padding-left: 50px;
		width: 60px;
		font-size:12px;
	}
	.list-body-box .el-icon-operation-delete {
		font-size:14px;
	}
	.list-body {
		width: calc(100% - 45px);
		padding-left: 25px;
		overflow: auto;
	}
	.list-body-title {
		color:#5A7B92;
		font-size: 12px;
		font-weight: bold;
		padding: 10px;
		border-bottom: 1px solid #F3F3F3;
	}
	.list-item-info {
		position: relative;
		padding: 10px;
		border-bottom: 1px solid #F3F3F3;
	}
	.list-item-info:hover{
		background:#EDF6FF;
	} 
	.list-item-op {
		padding: 5px;
		display: inline-block;
		position: absolute;
		top: 0px;
		right: 5px;
		color: red;
		cursor: pointer
	}
	.el-pagination .el-select .el-input{
		width:85px;
	}
	.el-pagination .el-select .el-input .el-input__inner{
		width:85px;
		background:#fff !important;
	}
	.overHide{
		overflow: hidden;	
	}
	.r70{
		right: 70px;	
	}
	.r10{
		right:10px
	}
	.h100{height: 100%}
	.ml10{
		margin-left: 10px
	}
	.curpo{
		cursor: pointer
	}
	.selectBox{
		flex: auto;
		padding: 10px;
		display: flex;
	}
	.selectBox .span2{
		font-size:16px;
		cursor: pointer;
	}
	.wenJianBox{
		height: 60%;
		display: flex;
		overflow: visible;
		border: 1px solid #d1ecf5;
	}
	.wenJianBoxPad{
		width: 260px;
		border-right: 1px solid #d1ecf5;
	}
	.list-item-info .el-icon-circle-close{
		position:absolute;
		top:9px;
		right:20px;
		font-size:18px;
		display:block;
	} 
	.linkbuttonGroup {
		margin-top:6px;
	}
	.msgNotify{
		word-wrap: break-word;
		word-break: break-all
	}
	.msgNotify .el-notification__title{
		margin-left: 5px
	}
	.settingBox{
		background: rgba(0,0,0,0.3);
	}
	.settingBoxForm .el-form-item__label{
		line-height: 24px
	}
	.setEnb{
		margin:30px
	}
	.setEnb .el-form-item__label{
		width: 150px
	}
	.pageFooter{
		height:55px;
		border-top:1px solid #EEEEEE;
		padding:15px 0 15px 40px;
		box-sizing:border-box;
		position: absolute;
		bottom:0px;
		width:100%;
	}
	.pageFooter .el-button, .footerBtn{
		width:86px;
		height:25px;
		border-radius:unset;
		padding:0px;
	}
	.tslide {
		right:0px;
		top:45px;
		box-shadow:0 0 50px rgba(158,200,222,0.35);
	}
	.noDateContainer{
		width:100%;
		height:100%;
		display:flex;
		justify-content:center;
		align-items:center;
	}
	.noDatePage{
		fant-family:'微软雅黑 Bold','微软雅黑 Regular','微软雅黑 ';
		font-weight:700;
		font-style:normal;
		font-size:18px;
		color:#666666;
	}
	.ml20{
		margin-left:20px
	}
	.f12{
		font-size: 12px
	}
	.chengong .el-icon-status-active:before{
		color:#67D972;
		font-size: 20px
	}
	.shibai .el-icon-status-active:before{
		color:#E88282;
		font-size: 20px
	}
	.el-icon-status-conn-off:before ,.el-icon-status-disable:before , .el-icon-status-enable:before{
		font-size: 20px
	}
	.ml5{margin-left:5px}
	.el-icon-circle-warning:before{
		font-size: 20px
	}
	.selectBox .el-input{
		width:120px;
		border:#4D84FF dashed 1px;
		margin-top:-8px
	}
	.el-icon-status-conn-on:berfoe{
		font-size: 20px
	}
	.userBox .el-form-item__error{
		margin-left:70px
	}
</style>

<!-- 虚拟CBSD监控页面  -->
<div class="panelDefault overHide" id='virtualSASPage' >
	<!-- SAS 切换CBSD和虚拟CBSD -->
	<div class="circleIcon placeholder-bt selectBox " style='margin-right:220px;'>
		<el-select v-model='selectList' @change='changeCbsd'>
			<el-option v-for="item in selectListData" :key="item.value" :value="item.value" :label="item.label"></el-option>
		</el-select>
	</div>
	<!-- SAS 操作按钮 -->
	<div v-show="true" class="circleIcon placeholder-bt"  placeholder="<%=rb.getString("TianJia")%>" style='margin-right:160px'>		
		<span class="el-icon-circle-add el-icon" @click="addconfig"></span>
	</div>
	<div  class="circleIcon placeholder-bt" placeholder="<%=rb.getString("DaoRu")%>"   style='margin-right:120px'>		
		<span class="el-icon-circle-import el-icon" @click="sasImport"></span>
	</div>
	<div  class="circleIcon placeholder-bt CODE_ADVANCE_SAS hidden" placeholder="<%=rb.getString("SheZhi")%>" style='margin-right:80px'>		
		<span class="el-icon-circle-setting el-icon" @click="sasSetting"></span>
	</div>
	<div  class="circleIcon placeholder-bt" placeholder="<%=rb.getString("RiZhi")%>" style='margin-right:40px'>		
		<span class="el-icon-circle-log el-icon" @click="sasALlLog"></span>
	</div>
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>">		
		<span class="el-icon-circle-export el-icon" @click="exportCSV"></span>
	</div>
	<select id="searchDeViceType"  name='deviceType' data-options="editable:false" style="display:none" class="">
		<option value="eNB">eNB</option>
		<option value="CPE">CPE</option>
	</select>
	<input id='deviceTypeSelect' style="display:none">
	<input id='tabNames' style="display:none">
	<!-- 主页面区域 -- tab页 -->
	<el-tabs v-model="activeName" @tab-click='tabClick' style="height:100%;">
		<!-- eNB -->
		<el-tab-pane :label="tabNameEnb" name="eNB">
			<el-ctable id="virtualeNBTable" ref="enbTable" :url="enbUrl" :time="refreshVirtualTime" @sort-change="sortChangeEnb"
				:row-key="'serialNumber'" :query-params="params_sasEnb">
				<!-- 高级查询 -- eNb-->
				<template slot="toolbar">
					<el-form :model="params_Enb" ref="params_Enb" label-position="top" inline=true>
						<el-query @query="queryEnb" @advance-query="advanceQueryEnb" @reset="resetQueryEnb" :ok-text="queryButton" 
							placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>" :reset-text="resetButton" :arrow-text="advanceText">
							<template slot="form">
								<el-form-item label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber">
									<el-input v-model="params_Enb.serialNumber"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("HostName")%>' prop="hostName">
									<el-input v-model="params_Enb.hostName"></el-input>
								</el-form-item>
								
							</template>
						</el-query>
					</el-form>
				</template>
					
				<!-- 主列表 -->
				<el-table-column label='' width="30" prop="">
					<template slot-scope="scope">
						<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" ></div>
					</template>
				</el-table-column>
					
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' min-width="150" prop="serialNumber" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("HostName") %>' min-width="150" prop="hostName" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai") %>' min-width="120" prop="state" >
					<template slot-scope="scope">
						<div v-if="scope.row.state == '0'">Unregistered</div>
						<div v-if="scope.row.state == '1'">Registered</div>
						<div v-if="scope.row.state == '3'">Granted</div>
						<div v-if="scope.row.state == '4'">Grant Suspended</div>
						<div v-if="scope.row.state == '5'">Authorized</div>
						<div v-if="scope.row.state == '6'">Transmission</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("SASLeiXing") %>' min-width="80" prop="category" ></el-table-column>
				<el-table-column label='<%=rb.getString("CBSDID") %>' min-width="180" prop="cbsdId" ></el-table-column>
				<el-table-column label='<%=rb.getString("ShouQuan") %>' min-width="150" prop="grants" ></el-table-column>
				<el-table-column label='<%=rb.getString("SASEIRP") %>' min-width="180" prop="maxEirp" ></el-table-column>
				<el-table-column label='<%=rb.getString("PinLv") %>' min-width="150" prop="frequency" ></el-table-column>
				<el-table-column label='<%=rb.getString("grantGuoQiShiJian")%>' min-width="150" prop="grantExpireTime" ></el-table-column>
				<el-table-column label='<%=rb.getString("transmitGuoQiShiJian")%>' min-width="150" prop="transmitExpireTime" ></el-table-column>
			</el-ctable>
			<el-cmenu ref="menuVirtualeNB" :data="menus" @click="clickMenu"></el-cmenu>
		</el-tab-pane>
		<!-- CPE -->
		<el-tab-pane :label="tabNameCpe" name="CPE">
			<el-ctable id="virtualCPETable" ref="cpeTable" :url="cpeUrl" :time="refreshVirtualTime" @sort-change="sortChangeCPE"
				:row-key="'serialNumber'" :query-params="params_sasCpe">
				<!-- 高级查询 -- CPE-->
				<template slot="toolbar">
					<el-form :model="params_Cpe" ref="params_Cpe" label-position="top" inline=true>
						<el-query @query="queryCpe" @advance-query="advanceQueryCpe" @reset="resetQueryCpe" :ok-text="queryButton" 
							placeholder="IMSI / <%=rb.getString("CPEName")%> " :reset-text="resetButton" :arrow-text="advanceText">
							<template slot="form">
								<el-form-item label='<%=rb.getString("CPEName")%>' prop="cpeName">
									<el-input v-model="params_Cpe.cpeName"></el-input>
								</el-form-item>
							</template>
						</el-query>
					</el-form>
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='' width="30" prop="">
					<template slot-scope="scope">
						<div class="el-icon el-icon-operation-more curpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose" ></div>
					</template>
				</el-table-column>
				
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' min-width="150" prop="serialNumber" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("CPEName") %>' min-width="150" prop="cbsdName" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai") %>' min-width="140" prop="state" >
					<template slot-scope="scope">
						<div>
							<span v-if="scope.row.cpeSasStatusEqualOmc == '0'" class="el-icon el-icon-circle-warning" ></span>
							<span v-if="scope.row.state == '0'">Unregistered</span>
							<span v-if="scope.row.state == '1'">Registered</span>
							<span v-if="scope.row.state == '3'">Granted</span>
							<span v-if="scope.row.state == '4'">Grant Suspended</span>
							<span v-if="scope.row.state == '5'">Authorized</span>
							<span v-if="scope.row.state == '6'">Transmission</span>
						</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("SASLeiXing") %>' min-width="80" prop="category" ></el-table-column>
				<el-table-column label='<%=rb.getString("ShouQuan") %>' min-width="150" prop="grants" ></el-table-column>
				<el-table-column label='<%=rb.getString("CBSDID") %>' min-width="180" prop="cbsdId" ></el-table-column>
				<el-table-column label='<%=rb.getString("LGWIPDiZhi") %>' min-width="180" prop="lgwIp" ></el-table-column>
				<el-table-column label='<%=rb.getString("SASEIRP") %>' min-width="180" prop="maxEirp" ></el-table-column>
				<el-table-column label='<%=rb.getString("PinLv") %>(MHz)' min-width="150" prop="frequency" ></el-table-column>
				<el-table-column label='<%=rb.getString("grantGuoQiShiJian")%>' min-width="150" prop="grantExpireTime" ></el-table-column>
				<el-table-column label='<%=rb.getString("transmitGuoQiShiJian")%>' min-width="150" prop="transmitExpireTime" ></el-table-column>
			</el-ctable>
			<el-cmenu ref="menuVirtualCPE" :data="menus" @click="clickMenu"></el-cmenu>
		</el-tab-pane>
		<el-tslide class='tslide' ref="tslide" :url="tsslideUrl" :title="slideTitleView" :footer="false" :header='tsslideHeader' :position="tsslidePosition"
					:height="tsslideHeight" :modal='modal'  :width="tsslideWidth" @cancel='cancelImport' >
		</el-tslide>
	</el-tabs>

	 <!-- 二级页面 -- insatall ,设置Enb和 过程  -->
	 <el-slide ref="slideView":url="slideUrl" :title="slideTitleView" :footer="false" :header='slideHeader' :position="slidePosition" :force-position="true"
	    :height="slideHeight" :modal='modal' :width="slideWidth" @cancel='cancelViewSlide'>
		<div class='setEnb settingBoxForm'>
				<el-form label-position="left" :model="settingsEnbForm" ref="settingsEnbForm" >
					<el-form-item  label="<%=rb.getString("SASZiDongZhuCeKaiGuan")%>" prop="sasAble" >
						<el-select v-model="settingsEnbForm.sasAble" :disabled="isabled">
							<el-option v-for="item in selectSasAble" :key="item.value" :value="item.value" :label="item.label"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item  label="<%=rb.getString("SASTiGongShang")%>" prop="sasProvider" >
						<el-select v-model="settingsEnbForm.sasProvider" :disabled="isabled">
							<el-option v-for="item in selectSasProvider" :key="item.value" :value="item.value" :label="item.label"></el-option>
						</el-select>
					</el-form-item>
				</el-form>
			</div>
			<div class="pageFooter">
				<el-button type="primary"  @click="submitSASSetting"><%=rb.getString("QueDing")%></el-button>
				<el-button  @click="cancelViewSlide"><%=rb.getString("QuXiao")%></el-button>
			</div>
	</el-slide>
	
</div>

<script type="text/javascript">
var virtualSASPage = new Vue({
	el:'#virtualSASPage',
	data(){
		return{
			showWindowInfo:false,
			dialogTitle:'',
			dialogUrl:'',
			windowWidth:'500px',
			tabNameEnb:'Virtual eNB',
			tabNameCpe:'Virtual CPE',
			selectList:"2",
			selectListData:[
				{
					value:'1',
					label:'CBSD'
				},
				{
					value:'2',
					label:'Virtual CBSD'
				}
			],
			activeName:'eNB',
			params_sasEnb:{
				timeZone: timeZone,
				deviceType:'eNB',
				search_text:''
			},
			params_sasCpe:{
				timeZone: timeZone,
				deviceType:'CPE',
				search_text:''
			},
			cpeUrl:'',
			enbUrl:'${ctx}/cell/SAS/getVirtualCbsdGridData.action',
			batchSeting:{
				setslideShow:false,
				height:'',
				modal:true,
				width:'800px'
			},
			settingsEnbForm:{ // 设置SAS开关
				sasAble:'',
				sasProvider:'',
			},
			selectSasAble:[
				{value:'1',label:'Enable'},
				{value:'0',label:'Disable'}
			],
			selectSasProvider:[
				{value:'0',label:'Federated Wireless'},
				{value:'2',label:'Amdocs'},
				{value:'3',label:'CommScope'},
				{value:'4',label:'Google'},
			],
			settingsInfo:{ // 设置详情页面的参数
				userId: '',
				callSign: ''
			},
			settingsRules:{
				userId:[
					{required:true,message:'<%=rb.getString("QingShuRu")%>User ID',trigger:'blur'},
				]
			},
			params_event:{
				timeZone:timeZone,
				id:'',
				neSerialNumber:'',
				eventName:'',
				neIpAddress:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			params_Enb:{
				serialNumber:'',
				hostName:'',
				cbsdId:'',
				sas_active_status:''
			},
			params_Cpe:{
				imsi:'',
				cpeName:'',
				macAddress:'',
				sas_active_status:''
			},
			activeAtatusData:[
				{
					value:'',label:'<%=rb.getString("QuanBu")%>'
				},
				{
					value:'1',label:'<%=rb.getString("JiHuo")%>'
				},
				{
					value:'0',label:'<%=rb.getString("QuJiHuo")%>'
				}
			],
			params: {
				searchText: '',
				device_name: '',
				operate_type: '',
				time: ''
			},
			paramsDevice: {
				searchText: ''
			},
			paramsAlarm: {
				searchText: ''
			},
			settingSlide:{},
			settingSlideHeight:'',
			settingSlideModal:'',
			settingSlideWidth:'',
			menus:[],
			serialNumber:'<%=rb.getString("SheBeiWeiYiBiaoZhi")%>',
			reDiv:true,
			rowData:[],
			selection:'',
			slideUrl:'',
			slideTitle:'',
			slideTitleView:'',
			advanceText:'<%=rb.getString("GaoJiChaXun")%>',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
			tsslideUrl:'',
			tsslideHeader:'',
			tsslideFooter:'',
			tsslidePosition:'',
			tsslideHeight:'',
			tsslideWidth:'',
			showFlag:false,
			height:'100%',
			width:'100%',
			modal:false,
			tabsShow: true,
			queryButton:'<%=rb.getString("ChaXun")%>',
			resetButton:'<%=rb.getString("ChaXunChongZhi")%>',
			tableRule:{
				eNB:'enbTable',
				CPE:'cpeTable',
			},
			buttomOptShow: false,
			selectedDeviceShow: false,
			selectedList: [{serialNumber:123}], //已选设备
			islocal: isLocal == 'true' && is_super_user == 'true',
			isabled:false,
			sortSaseNB:'',
			orderSaseNB:'',
			sortSasCPE:'',
			orderSasCPE:'',
			refreshVirtualTime:6
		}
	},
	computed: {
		selectedTitle(){
			return '<%=rb.getString("XiaoZhanBianMa")%>';
		},
		bottomSliderCls() {
			var vm = this;
			return {
				'bottom-slider': true,
				show: vm.buttomOptShow
			};
		},
		selectedDevicesInfoCls() {
			return {
				'selected-devices-info': true,
				show: this.selectedDeviceShow
			};
		},
		controlClass(){
			if(this.tabsShow && writableMap['CODE_ADVANCE_SAS']){ // 虚拟CBSD添加权限
				return true
			}else{
				return false
			}
		}
		
	},
	mounted(){
		this.init();
		eventBus.$off('close-dialog').$on('close-dialog',this.cancelViewSlide); 
		eventBus.$off('close-dialogs').$on('close-dialogs',this.cancelImport); 

		
	},
	watch:{
		// 监听筛选
		selection(){
			var activeName = this.$root.activeName;
			var data;
			if(activeName == 'eNB'){	    		
				data = this.selection;
	    	}else if(activeName == 'CPE'){
	    		data = this.$refs.cpeTable.getData();
	    	}
			
			 
			var cellCodes = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.serialNumber + ","
				})
			}
			
		},
	},
	methods:{
		init(){
			var vm = this;
			axios.post('${ctx}/cell/SAS/getSasBillingDeviceEnable.action').then(function(response){
				var data = response.data
				if(data.sasBillingDeviceEnable){ // // 接口返回true 且 用户是admin 的时候可以修改
					if(is_super_user === 'true'){
						vm.isabled = false
					}else{
						vm.isabled = true
					}
				}else{
					vm.isabled = false
				}
			})
			
			$('.el-loading-mask').css('display','none')
			$('#tabNames').val('2')
		},
		closeDialog(){
			var vm = this;
			vm.showWindowInfo = false;
			vm.dialogUrl = ''
		},
		// 弹窗打开成功传递参数  
		openDialogSuc(){
			eventBus.$emit('open-dialog');
		},
		handerClose(){ //点击页面其他地方菜单收起
	        this.$refs.menuVirtualeNB.hide();
	        this.$refs.menuVirtualCPE.hide();
	    },
		/**
		 * 更多操作
		 * @parame row:1.install params  2.过程  3.删除  
		*/
		optClick(row,ev){ 
	    	var vm = this,
				disableFlag = false,
				notUserFlag = false,
				installFlag = false,
			    activeName = this.$root.activeName;
	    	
			if(row.connection_status == "Off"){ //判断开关 和 install params 禁用与否 当基站不在线时禁用
				notUserFlag = true;
				installFlag = true;
			}
	    	vm.menus= [
		    	{label:'Install Params',cls:"el-icon el-icon-operation-info CODE_ADVANCE_SAS hidden",code:'view',show:true,disable:installFlag},
				{label:'<%= rb.getString("SASProcedure")%>',cls:"el-icon el-icon-operation-procedure",code:'procedure',show:true},
				{label:'<%= rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',show:true},
		         
		    ]
			vm.rowData = row;
	    	vm.$nextTick(function(){
	    		document.body.click();
				if(activeName == 'eNB'){	    		
					vm.$refs.menuVirtualeNB.show(ev);
		    	}else if(activeName == 'CPE'){
		    		vm.$refs.menuVirtualCPE.show(ev);
		    	}
	    	});
	    },
		/**
		 *  获取点击项数据
		 * @parame ev:点击属性数据
		*/
	    clickMenu(ev){ //
	    	var codes = {
	    		view:this.goProperties, // 跳转到 install params
	    		procedure:this.goProduce,// 跳转到过程
				del:this.deleteVirtual
	    	};
	    
	    	if(codes[ev.code]){
				codes[ev.code](this.$root.rowData,ev.code)
	    	}
	    },
		/**
		 *  install params
		*/
	    goProperties(row){ 
			var vm = this;
			var activeName = vm.$root.activeName,
				sn = row.serialNumber,
				deviceType = row.deviceType;
			vm.$root.slideUrl = '${ctx}/cell/SAS/toInstallParamPage.action?serialNumber='+sn+'&deviceType='+deviceType;
			vm.$root.slideTitleView = 'Install Params';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = true;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('open-dialog','edit',activeName,vm.rowData,vm.tabsShow);
			});
	    },
		/**
		 *  跳转到过程
		*/
	    goProduce(row){
	    	var vm = this ;
			var activeName = this.$root.activeName;
			vm.$root.slideUrl = '${ctx}/cell/SAS/toSasProcedure.action?serialNumber='+row.serialNumber;
			vm.$root.slideTitleView = '<%=rb.getString("SASProcedure")%>';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = true;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('open-dialog',vm.rowData,activeName,'virtualCbsd');
			});
	    },
	    deleteVirtual(row,code){
	    	var vm = this,
	    		url ='${ctx}/cell/SAS/deleteVirtualCbsd.action',
	    		confirmMsg = '<%=rb.getString("QueDingShanChuXuNiCBSD")%>',
	    		params = {
					serialNumber:row.serialNumber,
					deviceType:vm.activeName
	    		};
	    	
	    	vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)).then(function(response){
					let data = response.data
					if(data.success){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.$refs[vm.tableRule[vm.activeName]].refresh();
					}else{
						vm.$message({
							message: data.message,
							type:'error',
						});
					}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
		exportCSV(){ // 导出文件
			var vm = this,params={},exportUrl='';
			var activeName = vm.$root.activeName;
			if(activeName == 'eNB'){
				params={
					search_text:vm.params_sasEnb.search_text,
					serialNumber:vm.params_Enb.serialNumber,
					hostName:vm.params_Enb.hostName,
					deviceType:activeName,
					timeZone: timeZone,
					order:vm.sortSaseNB,
					sort:vm.orderSaseNB,
				}
		    }else if(activeName == 'CPE'){
		    	params={
					search_text:vm.params_sasCpe.search_text,
					imsi:vm.params_Cpe.imsi,
					cpeName:vm.params_Cpe.cpeName,
					deviceType:activeName,
					timeZone: timeZone,
					order:vm.sortSasCPE,
					sort:vm.orderSasCPE,
				}
		    }
			exportUrl ="${ctx}/cell/SAS/exportVirtualCbsdGridData.action";
			exportByForm(exportUrl,params);
		},
		addconfig(){ // 新建CBSD
			var vm = this;
			var activeName = this.$root.activeName;
			vm.$root.slideUrl = "${ctx}/cell/SAS/toInstallParamPage.action?serialNumber=120200005116A8P0145&deviceType=enb&_=1599708934179";
			vm.$root.slideTitleView = '<%=rb.getString("TianJia")%> CBSD';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideWidth = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = true;
			var iscbsd = true;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('open-dialog','add',activeName,vm.rowData,iscbsd);
			});
		},
		sasALlLog(){// 日志
			var vm = this;
			vm.$root.slideUrl = "${ctx}/cell/SAS/toSasMainLog.action";
			vm.$root.slideTitleView = '';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideWidth = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = true;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('open-dialog',vm.rowData);
			});
		},
		sasSetting(){// 设置
			var vm = this;
			axios.post('${ctx}/cell/SAS/getSasSettings.action').then(function(response){
				var data = response.data
				vm.settingsEnbForm.sasAble = data.sasEnable;
				vm.settingsEnbForm.sasProvider = data.sasProvider;
				var providers=[];
				var urls = data.sasURL;
				if(urls && urls.length){
					var sasUrlList = {};
					urls.map((item,index) => {
						sasUrlList[item.id] = item.url;
						providers.push({
							value: item.id,
							label: item.providerName
						})
					})
				}
				vm.selectSasProvider = providers
			})
			vm.$root.slideUrl = "";
			vm.$root.slideTitleView = '<%=rb.getString("SheZhi")%>';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideWidth = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = true;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('open-dialog',vm.rowData);
			});
		},
		submitSASSetting(){//sas设置提交事件
			var vm = this, url='${ctx}/cell/SAS/saveSasSettings.action',params={};
			params.sasEnable = vm.settingsEnbForm.sasAble;
			params.sasProvider = vm.settingsEnbForm.sasProvider;
			
			vm.$refs.settingsEnbForm.validate((r)=>{
				if(r){
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
						vm.cancelViewSlide()
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					})
				}
			})
			
		},
		sasImport(){ // 导入
			var vm = this;
			var activeName = this.$root.activeName;
			vm.$root.tsslideUrl = "${ctx}/cell/SAS/toImportInstallParam.action";
			vm.$root.slideTitleView = '<%=rb.getString("DaoRu")%> CBSDs';
			vm.$root.tsslidePosition = '';
			vm.$root.tsslideHeight = '440px';
			vm.$root.tsslideWidth = '740px';
			vm.$root.modal = false;
			vm.$root.tslideHeader = true;
			
			vm.$refs.tslide.showSlide(function(){
					eventBus.$emit('hander-rows',activeName)
				});	
		},
	    cancelViewSlide(){ // 二级页面关闭
	    	var vm = this;
	    	vm.$refs.slideView.hide();
	    },
		cancelImport(){ // 关闭导入页面
	    	var vm = this;
	    	 var obj = Vue.getInstance("#viewPciTaskEnb");
	         obj && document.querySelector("#viewPciTaskEnb").remove();
			vm.$refs.tslide.hide();
			vm.$refs[vm.tableRule[vm.activeName]].refresh();
		},
		//eNb-模糊查询
		queryEnb:function(val){
			this.params_sasEnb.search_text  = val;
			this.$refs.enbTable.refresh()
		},
		
		//eNb-高级查询
		advanceQueryEnb:function(){
			var vm = this,
			params = vm.params_sasEnb;
			params.search_text = vm.params_sasEnb.search_text;
			params.serialNumber = vm.params_Enb.serialNumber;
			params.cbsdName = vm.params_Enb.hostName;
			params.cbsdId = vm.params_Enb.cbsdId;
			params.opStatus = vm.params_Enb.sas_active_status;
			vm.$refs.enbTable.refresh();	

		},
		//eNb-重置
		resetQueryEnb(){
			var vm = this;
			vm.params_sasEnb.search_text = '';
			vm.$refs.params_Enb.resetFields();
		},
		// Cpe ——模糊查询
		queryCpe:function(val){
			this.params_sasCpe.search_text  = val;
			this.$refs.cpeTable.refresh()
		},
		// Cpe-高级查询
		advanceQueryCpe:function(){
			var vm = this,
			params = vm.params_sasCpe;
			params.search_text = vm.params_sasCpe.search_text;
			params.imsi = vm.params_Cpe.imsi;
			params.cbsdName = vm.params_Cpe.cpeName;
			params.macAddress = vm.params_Cpe.macAddress;
			vm.$refs.cpeTable.refresh();	
		},
		// Cpe-重置
		resetQueryCpe(){
			var vm = this;
			vm.params_sasCpe.search_text = '';
			vm.$refs.params_Cpe.resetFields();
		},
	    changeCbsd(value){
			var vm= this;
			vm.handerClose()
			if(value !== 1){ // 真实CBSD
				vm.$root.slideUrl = '${ctx}/cell/SAS/toSasMonitor.action';
				vm.$root.slidePosition = 'top';
				vm.$root.slideHeight = '100%';
				vm.$root.slideFooter = false;
				vm.$root.slideHeader = false;
				vm.$root.$refs.slideView.showSlide(function(){
					//eventBus.$emit('open-dialog');
				});
				vm.refreshVirtualTime = 0;
			}else{
				vm.tabNameEnb = 'eNB'
				vm.tabNameCpe ='CPE'
			}
		},
	    tabClick(tab){ // 列表的点击
	    	var vm = this;
	    	vm.cancelViewSlide();
			var activeName = this.$root.activeName;
			$('#deviceTypeSelect').val(activeName)
			if(activeName == 'eNB'){
				vm.resetQueryEnb()
				vm.enbUrl='${ctx}/cell/SAS/getVirtualCbsdGridData.action'
			}else{
				vm.resetQueryCpe()
				vm.cpeUrl='${ctx}/cell/SAS/getVirtualCbsdGridData.action'
			}
	    },
		sortChangeEnb(data){ //排序点击
			var vm =this,
				order={
					ascending:'asc',
					descending:'desc'
				};
			vm.sortSaseNB = data.prop;
			vm.orderSaseNB = order[data.order];		
		},
		sortChangeCPE(data){ //排序点击
			var vm =this,
				order={
					ascending:'asc',
					descending:'desc'
				};
			vm.sortSasCPE = data.prop;
			vm.orderSasCPE = order[data.order];		
		}
	}
})
</script>