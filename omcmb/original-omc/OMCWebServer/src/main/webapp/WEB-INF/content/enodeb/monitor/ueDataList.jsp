<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>

	span[class*='status_']{
		padding-left:25px;
	}
	.ups-icon-span span{
			display: inline-block;
			margin-top: -5px;
		}
		.title-text:after{
			display: none
		}
		.group-title{
			margin:10px
		}
		.el-card__body{
			display:flex;
			flex-direction:column;
			flex:1 1 auto;
			overflow:auto;
			padding: 0;
		}
		.footer{
			width: 100%;
			overflow: overlay;
			border-top: #EEEEEE solid 1px;
			position: absolute;
			bottom: 0px;
			height: 48px;
			line-height: 48px;
			
		}
		.footer .linkbutton{
			margin-left: 48px
		}
		.form-wrap{
			padding: 0 15px !important
		}
		.ml20{
			margin-left:20px
		}
		.plr15{
			padding: 0 15px !important
		}
		.plr15 .el-form-item__label{
			width:100px;
			float:left;
		}
		.el-form-item{
			margin-bottom:22px
		}
		#falutListPage{
			position: relative;
			overflow: hidden;
			height: 100%;
		}
		.h100{
			height:100%
		}
		.ml10{
			margin-left:10px
		}
		.curpo{
			cursor:pointer
		}
		.w290{
			width:290px
		}
		
		.mr5{
			margin-right:5px
		}
		.linkbuttonGroup{
			margin-left:50px
		}
		.el-form-item:after{
			display:none !important
		}
		.form-group .el-form-item__label{
			line-height: 28px
		}
		.texc{
			text-align: center
		}
		.ruo .el-icon-signal:before{
			color:#FF7B7B;
			font-size:20px
		}
		.zhengchang .el-icon-signal:before{
			color:#FFBB00;
			font-size:20px
		}
		.qiang .el-icon-signal:before{
			color:#67D972;
			font-size:20px
		}
		.log .el-icon-circle-close:before{
			font-size: 25px
		}
		.bottom-slider.show {
		bottom: 0px;
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
	.selectBox{
		flex: auto;
		padding: 10px;
		display: flex;
	}
	.selectBox .span2{
		font-size:16px;
		cursor: pointer;
	}
	.list-item-info .el-icon-circle-close{
		position:absolute;
		top:9px;
		right:20px;
		font-size:18px;
		display:block;
	} 
	.cpeLwaKai{
		display: inline-block;
		margin-top:2px;
		width:25px;
		height:25px;
		background:url(${ctx}/css/images/newIcon/statusIcon/TURBOkai.png) no-repeat;
	}
	.cpeLwaGuan{
		display: inline-block;
		margin-top:2px;
		width:25px;
		height:25px;
		background:url(${ctx}/css/images/newIcon/statusIcon/TURBOguan.png) no-repeat;
	}
	.cpeLwaWu{
		display: inline-block;
		margin-top:2px;
		width:25px;
		height:25px;
		background:url(${ctx}/css/images/newIcon/statusIcon/TURBOwu.png) no-repeat;
	}
	.f12{
		font-size:12px
	}
	.zhanweiBox{
		display: inline-block;
    height: 32px;
    padding: 0 15px;
    /* border: 1px solid #E9E9E9; */
    border-radius: 4px;
    margin-left: 20px;
    vertical-align: middle;
	}
	.no-data:before{
		content: ''
	}
	.slidrBox .el-card__body{
		border:none
	}
	#falutListPage .cpeCountTitle {
		font-size: 16px;
		font-weight: bold;
		color: rgba(0, 0, 0, 0.8);	
		height: 48px;
		line-height: 48px;
		padding: 0 20px;
	}
</style>

<div id="falutListPage" class='commonWarp'>
	
	<!-- 按钮 url="${ctx}/cell/ap/ueDataList.action" -- 导出 -->
	<div v-if="false"  class="circleIcon placeholder-bt" placeholder="<%=rb.getString("UEShu")%>" style="margin-right:60px;">		
		<span class="el-icon-circle-log el-icon" @click='goLog'></span>
	</div>
	<div  class="circleIcon placeholder-bt">		
		<span class="el-icon-circle-close el-icon" @click='closeSildeUeData'></span>
	</div>
	<div class='cpeCountTitle'>{{titleTab}}</div>
	
	<el-ctable id="activeFaultTable" ref="lwaList"  :height="height"  url="${ctx}/cell/ap/ueDataList.action" :row-key="'CPE_CODE'"
		 :query-params="params_ueData" :page-size="pageSize" pagination="true"  @selection-change='batchSelect'>
		
		<!-- 主列表 -->
		<el-table-column v-if="isLWAEnable" label='' width="50" type="selection" prop="ck" :selectable='disableNoTurbo'></el-table-column>
		<el-table-column v-if="isLWAEnable" label='' width="30" prop="">
				<template slot-scope="scope">
					<div v-if="scope.row.CAPABILITY=='0'" class="el-icon el-icon-operation-more disabled curpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose" :disabled="false"></div>
					<div v-else class="el-icon el-icon-operation-more curpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose" :disabled="false"></div>
				</template>
		</el-table-column>
		
		<el-table-column label='IMSI' min-width="130" prop="IMSI" show-overflow-tooltip ></el-table-column>
		<el-table-column label='<%=rb.getString("CPEName")%>' min-width="130" prop="CPE_NAME" show-overflow-tooltip ></el-table-column>
		<el-table-column label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="130" prop="MODEL_NAME" show-overflow-tooltip >
			<template slot-scope="scope">
				<div v-if="scope.row.CAPABILITY=='0' && isLWAEnable" class="ruo">
					<span class='cpeLwaWu' style='font-size:22px'>
						<span style='font-size:12px;margin-left:25px'>{{scope.row.MODEL_NAME}}</span>
					</span>
				</div>
				<div v-if="scope.row.CAPABILITY=='1' && scope.row.LTE_TURBO_ENABLE=='1' && isLWAEnable">
					<span class='cpeLwaKai' style='font-size:22px'>
						<span style='font-size:12px;margin-left:25px'>{{scope.row.MODEL_NAME}}</span>
					</span>
				</div>
				<div v-if="scope.row.CAPABILITY=='1' && scope.row.LTE_TURBO_ENABLE=='0' && isLWAEnable">
					<span class='cpeLwaGuan' style='font-size:22px'>
						<span style='font-size:12px;margin-left:25px'>{{scope.row.MODEL_NAME}}</span>
					</span>
				</div>
				<div v-if="!isLWAEnable">
					<span style='font-size:12px;'>{{scope.row.MODEL_NAME}}</span>
				</div>
			</template>
		</el-table-column>
		<!--<el-table-column label='RAN' min-width="130" prop="ran" show-overflow-tooltip ></el-table-column>-->
		<el-table-column label='<%=rb.getString("MACDiZhi") %>' min-width="130" prop="MACADDRESS" show-overflow-tooltip ></el-table-column>
		<el-table-column label='<%=rb.getString("IPDiZhi") %>' min-width="130" prop="IPADDRESS" show-overflow-tooltip ></el-table-column>
		<el-table-column label='UL_MCS' min-width="130" prop="UL_MCS" show-overflow-tooltip ></el-table-column>
		<el-table-column label='DL_MCS' min-width="130" prop="DL_MCS" show-overflow-tooltip ></el-table-column>
		<el-table-column label='SINR' min-width="130" prop="CPE_SINR" show-overflow-tooltip ></el-table-column>
		<el-table-column label='RSRP1' min-width="130" prop="RSRP0" show-overflow-tooltip >
			<template slot-scope="scope">
				<div v-if="scope.row.RSRP0<lowVal" >
					<span class="el-icon el-icon-signal signal-low">
						<snap class="f12">{{scope.row.RSRP0}}</span>
					</span>
				</div>
				<div v-if="scope.row.RSRP0>highVal" >
					<span class="el-icon el-icon-signal signal-high">
						<snap class="f12">{{scope.row.RSRP0}}</span>
					</span>
				</div>
				<div v-if="scope.row.RSRP0<highVal &&scope.row.RSRP0>lowVal ">
					<span class="el-icon el-icon-signal signal-normal">
						<snap class="f12">{{scope.row.RSRP0}}</span>
					</span>
				</div>
			</template>
		</el-table-column>
		<el-table-column label='RSRP2' min-width="130" prop="RSRP1" show-overflow-tooltip >
			<template slot-scope="scope">
				<div v-if="scope.row.RSRP1<lowVal" >
					<span class="el-icon el-icon-signal signal-low">
						<span class="f12">{{scope.row.RSRP1}}</span>
					</span>
				</div>
				<div v-if="scope.row.RSRP1>highVal" >
					<span class="el-icon el-icon-signal signal-high">
						<span class="f12">{{scope.row.RSRP1}}</span>
					</span>
				</div>
				<div v-if="scope.row.RSRP1<highVal &&scope.row.RSRP1>lowVal " >
					<span class="el-icon el-icon-signal signal-normal">
						<span class="f12">{{scope.row.RSRP1}}</span>
					</span>
				</div>
			</template>
		</el-table-column>
	</el-ctable>
	<el-cmenu ref="menuActiveFault" :data="menus" @click="clickMenu" ></el-cmenu>
			
	<!-- 详情-->
	 <el-slide ref="slider" :url="slideUrl" class="slidrBox"
	 :title="slideTitle" :footer="footerShow" :header='headerShow' :position="slidePosition" :modal="true" :height="sliderHeight" :width="sliderWidth"  @cancel='cancelSlide'>
		<div class="log">
			<div  class="circleIcon placeholder-bt" placeholder="<%=rb.getString("GuanBi")%>"style="margin-right:40px;">		
				<span class="el-icon-circle-close el-icon" @click='closeSilde'></span>
			</div>
			<el-tabs v-model="logParams.activeName" class="h100" >
				<el-tab-pane name="activeFault" label='<%=rb.getString("UEShu")%>'> 
					<el-ctable id="activeFaultTable" ref="lwaLogList" :time="6" :height="logParams.height" :url="silderUrl"
						:query-params="logParams.params_lwa" :page-size="logParams.pageSize" pagination="true" :rownumber=true >
						<template slot="toolbar">
							<div class="zhanweiBox">
							</div>
						</template>
						<!-- 主列表 -->
						<el-table-column v-for="col in hisColumns" :label='col.title' :min-width="col.width" :prop="col.field"></el-table-column>
					</el-ctable>
				</el-tab-pane>
			</el-tabs>
		</div>
	 </el-slide>
	<!-- 二级页面 -- 批量操作 -->
	<div :class="selectedDevicesInfoCls">
		<div class="enb-list-ctn">
			<div class="list-title">
				<%=rb.getString("YiXuanSheBei")%>
				<span  @click="selectedDeviceShow=false"><i class="el-icon el-icon-close"></i></span>
			</div>
			<div class="list-body-box" >
				<div class="list-body-title">
					{{selectedTitle}}
					<div class="operation_deletes el-icon el-icon-operation-delete" title="Delete All"  @click="delAllSelectedRecord"> <%=rb.getString("QingKong")%></div>
				</div>
			</div>
			<div class="list-body" style="flex: auto;">
				<div v-for="item in selectedList"
					 class="list-item-info">
					 	{{item.serialNumber||'.'}}
					 	<span class="el-icon el-icon-circle-close" @click="delSingleRecord(item)"></span>
				  </div>
			
			</div>
		</div>
	</div>
	<div :class="bottomSliderCls">
		<div style="display: flex;">
			<div class='selectBox'>
				<%=rb.getString("YiXuanSheBei")%>（<span id="device_count" style="color: blue;">{{selectedList.length}}</span>）
				<span id="sliderArrow" class='el-icon span2' :class="{'el-icon-circle-down': !selectedDeviceShow, 'el-icon-circle-up':selectedDeviceShow}"  @click="selectedDeviceShow = !selectedDeviceShow"> </span>
			</div>
			<div>
				<div class="linkbuttonGroup">
					<el-button type="primary" size="small" style="margin-right:-15px" @click="batchCbrsas('1')">LTE-TURBO <%=rb.getString("QiYong")%></el-button>
					<el-button type="" size="small" @click="batchCbrsas('0')">LTE-TURBO <%=rb.getString("JinYong")%></el-button>
					<el-button size="small" @click="cancelBatchOpt"><%=rb.getString("QuXiao")%></el-button>
					
				</div>
			</div>
		</div>
	</div>
</div>

<script>
	var faultListVue = new Vue({
		el:'#falutListPage',
		data(){
			var vm = this,
				is436q = window.sessionStorage.getItem('is436Q'),
				columns = [
					{ field:'ue_id',title:'ueId',width:100,align:'center'},  
					{ field:'imsi',title:'imsi',width:120,align:'center'},	  
					{ field:'vmac',title:'vmac',width:100},
					{ field:'downlink_rate',title:'<%=rb.getString("XiaXingTunTuLv")%>'},
					{ field:'uplink_rate',title:'<%=rb.getString("ShangXingTunTuLv")%>'},
					{ field:'ip',width:100,title:'<%=rb.getString("IPDiZhi")%>'},
					{ field:'port',width:80,title:'<%=rb.getString("DuanKou")%>'},
					{ field:'ulsinr',width:100,title:'<%=rb.getString("ShangXingSinr")%>'},
					{ field:'dlcqi',width:100,title:'<%=rb.getString("XiaXingCqi")%>'},
					{ field:'ulmcs',width:100,title:'<%=rb.getString("ShangXingmcs")%>'},
					{ field:'dlmcs',width:100,title:'<%=rb.getString("XiaXingmcs")%>'},
					{ field:'txpower',width:100,title:'<%=rb.getString("FaSongGongLv")%>(dBm)'},
					{ field:'uplink_bler',width:100,title:'<%=rb.getString("ShangXingbler")%>(%)'},
					{ field:'downlink_bler',width:100,title:'<%=rb.getString("XiaXingbler")%>(%)'},
					{ field:'pathloss',width:100,title:'<%=rb.getString("LuJingSunHao")%>(dBm)'},
				];
				if(is436q == 'true') {
					columns = [
						{ field:'ue_id',title:'ueId',width:100,align:'center'},  
						{ field:'imsi',title:'imsi',width:120,align:'center'},	  
						{ field:'vmac',title:'vmac',width:100},
						{ field:'downlink_rate',title:'<%=rb.getString("XiaXingTunTuLv")%>'},
						{ field:'uplink_rate',title:'<%=rb.getString("ShangXingTunTuLv")%>'},
						{ field:'ip',width:100,title:'<%=rb.getString("IPDiZhi")%>'},
						{ field:'port',width:80,title:'<%=rb.getString("DuanKou")%>'},
						{ field:'ulsinr',width:100,title:'<%=rb.getString("ShangXingSinr")%>'},
						{ field:'p_dlcqi',width:100,title:'P_Dlcqi'},
						{ field:'s_dlcqi',width:100,title:'S_Dlcqi'},
						{ field:'ulmcs',width:100,title:'<%=rb.getString("ShangXingmcs")%>'},
						{ field:'p_dlmcs',width:100,title:'P_Dlmcs'},
						{ field:'s_dlmcs',width:100,title:'S_Dlmcs'},
						{ field:'txpower',width:100,title:'<%=rb.getString("FaSongGongLv")%>(dBm)'},
						{ field:'uplink_bler',width:100,title:'<%=rb.getString("ShangXingbler")%>(%)'},
						{ field:'p1_downlink_bler',width:100,title:'P_TB1_Downlink_BLER(%)'},
						{ field:'p2_downlink_bler',width:100,title:'P_TB2_Downlink_BLER(%)'},
						{ field:'s1_downlink_bler',width:100,title:'S_TB1_Downlink_BLER(%)'},
						{ field:'s2_downlink_bler',width:100,title:'S_TB2_Downlink_BLER(%)'},
						{ field:'pathloss',width:100,title:'<%=rb.getString("LuJingSunHao")%>(dBm)'},
					];
				}
			
			return{
				hisColumns: columns,
				selection:'',
				searchText:'',
				activeName:'ueList',
				height:'100%',
				pageSize:20,
				menus:[],
				modal:true,
				width:'800px',
				timeZone:timeZone,
				ueUrl:'',
				params_ueData:{ // 初始化数据参数
					smallCellCode: window.sessionStorage.getItem('ueSmallCellCode'),
					operatorCode:operator_code,
					eci:window.sessionStorage.getItem('ueEci'),
					pci:window.sessionStorage.getItem('uePci'),
					earfcn:window.sessionStorage.getItem('ueEarfcn')
				},
				deviceType:[],
				rowData:[],
				slideUrl:'',
				slideTitle:'',
				sliderHeight: '100%',
		    	sliderWidth: '100%',
		    	footerShow: true,
		    	headerShow: true,
				slidePosition: 'top',
				snNumber:'111',
				titleTab:'',
				selectedList: [{serialNumber:123}], //已选设备
				excTaskMap: {}, // 记录异常日志已选数据
				selectedDeviceShow: false,
				buttomOptShow: false,
				url:"",
				lowVal:'',
				highVal:'',
				logParams:{
					height:'100%',
					pageSize:20,
					activeName:'activeFault',
					timeZone:timeZone,
					params_lwa:{ // 初始化数据参数
						enb_code: '',
					},
					rowData:[],
				},
				silderUrl:'',
				listdata:[
					{
						CPE_CODE:'235656',
						CAPABILITY:'1',
						IMSI:'65555',
						CAPABILITY:'0'
					},
					{
						CPE_CODE:'66666',
						CAPABILITY:'1',
						IMSI:'65555',
						CAPABILITY:'1'
					}
				],
				isLWAEnable: isLWAEnable
			}
			
		},
		methods:{
			init(){
				var vm = this;
				var smallCellCode = window.sessionStorage.getItem('ueSmallCellCode')
				var eci = window.sessionStorage.getItem('ueEci')
				var pci = window.sessionStorage.getItem('uePci')
				var earfcn = window.sessionStorage.getItem('ueEarfcn')
				vm.params_ueData.smallCellCode = smallCellCode;
				vm.params_ueData.eci = eci;
				vm.params_ueData.pci = pci;
				vm.params_ueData.earfcn = earfcn;
				vm.params_ueData.operatorCode = operator_code;
				vm.ueUrl=''
				vm.logParams.params_lwa.enb_code =  smallCellCode;
				vm.snNumber = window.sessionStorage.getItem('snNumber')
				var cellName =  window.sessionStorage.getItem('cellName')
				vm.titleTab ='<%=rb.getString("CPELianJieShu")%>(<%=rb.getString("XiaoZhanBianMa")%>:' + vm.snNumber  + ','+ '<%=rb.getString("HostName")%>:'+cellName + ')'
				//RSRP
				vm.lowVal = localStorage.getItem("rsrp1");
				vm.highVal = localStorage.getItem("rsrp2");
			},
			// 搜索函数
			searchResult(){
				let vm = this

				this.params_ueData.searchText = vm.searchText;
				this.$refs["lwaList"].refresh();	 
		   	},
			handleClick(){
			},
			handerClose(){ //点击页面其他地方菜单收起
		        this.$refs.menuActiveFault.hide();
		    },
			disableNoTurbo(row,index){
				if(row.CAPABILITY === '0'){
					return false;
				}else{
					return true;
				}
			},
			optClick(row,ev){ // 操作项： 1.开关
		    	var vm = this,
		    		activeName = vm.activeName;
			
				var showGuan = false;
				var showKai = false;
				
				
				if(row.LTE_TURBO_ENABLE== '0' && row.CAPABILITY!='0'){ // turbo为关闭状态 显示开按钮
					showKai = true
				}
				if(row.LTE_TURBO_ENABLE== '1' && row.CAPABILITY!='0'){ // turbo为开启状态 显示关按钮
					showGuan = true
				}
				var disabledShow = false;
				
		    	vm.menus= [
					{label:'LTE-TURBO <%= rb.getString("BuKeYong")%>',cls:"el-icon el-icon-operation-enable1",show:showGuan,code:'on'},
					{label:'LTE-TURBO <%= rb.getString("KeYong")%>',cls:"el-icon el-icon-operation-disable1",show:showKai,code:'off'},
				]
		    	this.rowData = row
		    	vm.$nextTick(function(){
		    		document.body.click();
					vm.$refs.menuActiveFault.show(ev);
					
		    	});
		    },
		    clickMenu(ev){ //操作项点击方法 
		    	var codes = {
					on:this.operLWA,
					off:this.operLWA
		    	}
		    	if(codes[ev.code]){
		    		codes[ev.code](this.$root.rowData,ev.code)
		    	}
		    },
			operLWA(row,status){ // 单独开关
					//1-开启 0-关闭
				var turboEnable = '';
				if(status == 'on'){
					turboEnable = '0'
				}else{
					turboEnable = '1'
				}
				var vm = this, confirmMsg='', message='',url='${ctx}/cell/ap/turboSetting.action';
				var params = {
							cpeCodes : row.CPE_CODE,
							turboEnable : turboEnable
					}
					
				if(status === 'on'){
					confirmMsg = '<%=rb.getString("PiLiangGuanCapacity")%>'
				}else{
					confirmMsg = '<%=rb.getString("PiLiangKaiCapacity")%>'
				}
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
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
						vm.$refs["lwaList"].refresh();
						}else{
							vm.$message({
								message:data.message,
								type:'error',
							})
						}
					}).catch(function(error){
						
					})
				}).catch()
			},
		
			cancelSlide(){
				this.$refs.slider.hide();
			},
		     cancelBatchOpt(){ // 取消批量操作
				var vm = this;
				vm.buttomOptShow = false;
				vm.selectedDeviceShow = false;
				vm.delAllSelectedRecord();
			},
			/**
			*  查看操作
			* @param row:当前数据
			*/
			cancelViewSlide(){ // 二级页面关闭
				var vm = this;
				vm.$refs.slideView.hide();
			},	
			/**
			 * 多选响应
			 * @param selection:选择的数据
			*/
			batchSelect(selection){ 
				var vm = this,
					tb = vm.$refs.lwaList;
				vm.selection = selection
	    		var cklist = tb.getChecked();
				setTimeout(function(){
					if(cklist && cklist.length) {
						vm.selectedList = cklist.map(function(value){
							var rowItem = {serialNumber: value};
							return rowItem;
						})
						vm.buttomOptShow = true;
					}else {
						vm.selectedList = [];
						vm.buttomOptShow = false;
						vm.selectedDeviceShow = false;
					}
				},10)
			},
			delAllSelectedRecord(){ // 清除当前tab的所有设备选择记录
				var vm = this;
				vm.$refs["lwaList"].clearSelection();
			},
			/**
			 * 清除当前tab的单个设备选择记录
			 * @param row:数据
			*/
			delSingleRecord(row) { 
				var vm = this,
					
					rowkey =  'serialNumber',
					tb = vm.$refs.lwaList,
					rows = tb.getData(),
					cklist = tb.getChecked();
				rows.map(function(item){
					if(item['CPE_CODE'] == row[rowkey]) tb.toggleRowSelection(item, false);
				});
				
				cklist.splice(cklist.indexOf(row[rowkey]),1);
				
				vm.selectedList = vm.selectedList.filter(function(item){
					return item[rowkey] != row[rowkey];
				});
			},	    
			selectChange(selection){ // 全选操作
				this.selection = selection // 获取选择的数据
			},
			batchCbrsas(statu){ //批量开关
				var vm = this,url='${ctx}/cell/ap/turboSetting.action';
				var selection = vm.selection
				var code = ''
				selection.map(function(item,idx){
					if(idx){
						code += ',';
					}
					code += item.CPE_CODE
				});
				let Msg = ''
				if(statu==="0"){
					Msg = "<%=rb.getString("PiLiangGuanCapacity")%>"
				}else{
					Msg = "<%=rb.getString("PiLiangKaiCapacity")%>"
				}
				var params = {
						cpeCodes : code,
						turboEnable : statu
				}
					vm.$confirm(Msg,'<%=rb.getString("QueRen")%>',{
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
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
						vm.$refs["lwaList"].refresh();
						}else{
							vm.$message({
								message:data.message,
								type:'error',
							})
						}
					}).catch(function(error){
						
					})
				}).catch()
			},
			goLog(){ // 跳转列表页
				var vm = this;
				vm.footerShow =false;
				vm.slideTitle = '';
				vm.slidePosition = 'top';
				vm.headerShow=false;
				vm.silderUrl="${ctx}/system/device/enb/uedata/getENBUeStatisticsDataList.action"
				vm.$refs.slider.showSlide(function(){
	    	    	
	    	    });
			},
			closeSilde(){ // 关闭弹窗
				this.$refs.slider.hide();
			},
			closeSildeUeData(){
				enbvm.closeCpeSlide();
			}

		},
		computed: {
			selectedTitle(){
				return 'CPE_CODE';
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
		},
		watch:{
			// 监听筛选
			selection(){
				var data = this.selection;
				var cellCodes = '';
				if(data.length != 0){
					data.map(function(item){
						cellCodes += item.CPE_CODE + ","
					})
				}
			},
		},
		mounted(){
			this.init()
		
		}
	});
</script>