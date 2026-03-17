<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
#accessControlDiv .queryInfo label{
	display:block;
	margin-bottom:5px;
	line-height:26px;
}
#accessControlDiv .queryInfo{
	display:inline-block;
	margin-right:80px;
}
#accessControlDiv .el-icon-status-disable:before{
	color:#C2C2C2;
}
#accessControlDiv .selected.el-icon-status-yes:before{
	color:#4D84FF;
	font-size:20px;
}
#accessControlDiv .noselected.el-icon-status-yes:before{
	color:#ECECEC;
	font-size:20px;
}
#accessControlDiv .el-icon-status-accept:before{
	color:#67D972;
}
#accessControlDiv .el-icon-status-reject1:before{
	color:#E88282;
}

#accessControlDiv .omcAccessControlDiv,#accessControlDiv .egwAccessControlDiv{
	height: 100%;
	display:flex;
	overflow:hidden;
	flex-direction:column;
}
#accessControlDiv .suc_count span,.fail_count span{
	height:28px;
	line-height:30px;
	padding: 0 10px;
	display:inline-block;
	vertical-align:bottom;
}
#accessControlDiv .suc_count,.fail_count{
	display:inline-block;
	
}
#accessControlDiv .suc_count span:first-child{
	border:1px solid #67D972;
	border-radius:4px 0px 0px 4px;
	border-right:0px;
	color:#67D972 !important;
	background:#EEFFF3;
}
#accessControlDiv .suc_count span:last-child{
	border:1px solid #67D972;
	border-radius:0px 4px 4px 0px;
	color:#333;
	font-weight:normal;
	margin-left: 0;
}
#accessControlDiv .fail_count span:first-child{
	border:1px solid #E88282;
	border-radius:4px 0px 0px 4px;
	border-right:0px;
	color:#E88282 !important;
	background:#FEF2F2;
}
#accessControlDiv .fail_count span:last-child{
	border:1px solid #E88282;
	border-radius:0px 4px 4px 0px;
	color:#333;
	font-weight:normal;
	margin-left: 0;
}
#accessControlDiv .suc_count i,.fail_count i{
	margin-right:5px;
	font-size:16px;
}
#accessControlDiv .egwMainBox{
	height: 100%;
	flex: 1;
	display: flex;
	flex-direction: column;
	position: relative;
}
#accessControlDiv .operationsBoxCls{
	position: absolute;
	right: 20px;
	top: 15px;
	z-index: 99
}
#accessControlDiv .headBtnCls .el-icon::before{
	font-size: 30px;
}
#accessControlDiv .egwDeviceTable{
	height: 100%;
}
#accessControlDiv .activeItemCls .el-icon::before{
	font-size: 16px;
	color: #67D972;
}
#accessControlDiv .noActiveItemCls .el-icon::before{
	font-size: 16px;
	color: #e88282;
	content:'\e6fb';
}
#accessControlDiv .enableExplain{
	padding-left: 5px;
	font-size:12px;
	color:#363B4E;
	font-weight: bold;
}
#accessControlDiv .maskBoxCls{
	position: absolute;
	left: 0;
	right: 0;
	top: 0;
	bottom:0;
	cursor: pointer;
}
#accessControlDiv .suc_count, .fail_count {
	height: 30px;
	line-height: 30px;
}
#accessControlDiv .infoTipPover { word-wrap:break-word; word-break: break-all;overflow: hidden; padding: 14px; line-height: 24px;}
</style>
<div class="overflow-cls">
	<div class='panelDefault' id='accessControlDiv' style="min-width: 1050px;overflow:hidden;">
		<div style='height:calc(100% - 2px);overflow:auto'>
			<div class="omcAccessControlDiv" v-show="neType !== 'EGW'">
				<div  style='flex:1;border-bottom:1px solid #E9E9E9;'>
					<div class="circleIcon placeholder-bt CODE_ADVANCE_ACCESS_CONTROL hidden" style="top: 15px;right:50px;" placeholder="<%=rb.getString("TianJia")%>">		
						<span class="el-icon el-icon-circle-add" @click='addRule'></span>
					</div>
					<div class="circleIcon placeholder-bt" style="top: 15px;right:10px;" placeholder="List">		
						<span class="el-icon el-icon-circle-taskList" @click='goList'></span>
					</div>
					<el-ctable id="accessRuleGrid" ref="ctableAccessRule" :url='ruleUrl' :height="height"
						:row-key="'id'" :query-params="params_accessRule" pagination="true" :rownumber=true>
						
						<!-- 模糊查询 -- 接入规则 -->
						<template slot="toolbar">
							<span style='font-size:14px;font-weight:bold;margin-left:20px;'><%=rb.getString("JieRuGuiZe")%></span>
							<div class='queryGroup'> 
								<el-input v-model='params_rule_form.searchText' @keyup.enter.native="queryRule" class='pairgrid-query' placeholder='<%=rb.getString("GuiZeMingCheng")%>'></el-input>
								<i @click='queryRule' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>
						</template>
			
						<!-- 主列表 -->
						<el-table-column label='' width="30" prop="" class-name="no-text-tips">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="optClickRule(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="300" prop="status">
							<template slot-scope="scope">
								<el-switch v-model="scope.row.status" active-color='#4D84FF' inactive-color='#DCDFE6' 
								active-value='1' inactive-value='0' :disabled="!omcOptBtnShow" @change="changeStatus($event,scope.row.id)"></el-switch>
								<span v-text="scope.row.status == 1 ? '<%=rb.getString("QiYong")%>' : '<%=rb.getString("JinYong")%>'"
								style='margin-left:5px;color:#333'></span>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("GuiZeMingCheng")%>' min-width="200"  prop="tempName" ></el-table-column>
						<el-table-column label='TAC' min-width="100" prop="controlTacEnable" >
							<template slot-scope="scope">
								<div v-if="scope.row.controlTacEnable == 0">
									<span class='el-icon el-icon-status-yes noselected'></span>
								</div>
								<div v-if="scope.row.controlTacEnable == 1">
									<span class='el-icon el-icon-status-yes selected'></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='ECGI' min-width="100" prop="controlEcgiEnable">
						<template slot-scope="scope">
								<div v-if="scope.row.controlEcgiEnable == 0">
									<span class='el-icon el-icon-status-yes noselected'></span>
								</div>
								<div v-if="scope.row.controlEcgiEnable == 1">
									<span class='el-icon el-icon-status-yes selected'></span>
								</div>
							</template></el-table-column>
						<el-table-column label='IP' min-width="100" prop="controlIpEnable" >
							<template slot-scope="scope">
								<div v-if="scope.row.controlIpEnable == 0">
									<span class='el-icon el-icon-status-yes noselected'></span>
								</div>
								<div v-if="scope.row.controlIpEnable == 1">
									<span class='el-icon el-icon-status-yes selected'></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("GPSWeiZhi")%>' min-width="100" prop="controlGpsEnable">
							<template slot-scope="scope">
								<div v-if="scope.row.controlGpsEnable == 0">
									<span class='el-icon el-icon-status-yes noselected'></span>
								</div>
								<div v-if="scope.row.controlGpsEnable == 1">
									<span class='el-icon el-icon-status-yes selected'></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("CaoZuoRen")%>' min-width="120" prop="updateUser"></el-table-column>
						<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' min-width="140" prop="update_time"></el-table-column>
					</el-ctable>
					<el-cmenu ref="menu_rule" :data="menus_rule" @click="clickMenuRule"></el-cmenu>
				</div>
				<div style='flex:1;overflow:hidden;position:relative'>
					<el-ctable id="accessControlGrid" ref="ctableAccessStatus" :url='statusUrl' :height="height" :time="6" 
						:row-key="'id'" :query-params="params_accessStatus" pagination="true" :rownumber=true 
						@load-success="loadSuccessStatus">
						
						<!-- 高级查询 -- 接入控制 -->
						<template slot="toolbar">
								<span style='font-size:14px;font-weight:bold;position:absolute;left:20px;top:16px;'><%=rb.getString("JieRuZhuangTai")%></span>
								<el-query @query="query" @advance-query="advanceQuery" @reset="resetQuery" ok-text="<%=rb.getString("ChaXun")%>" 
								reset-text="<%=rb.getString("ChaXunChongZhi")%>" placeholder='<%=rb.getString("GuiZeMingCheng")%>' 
								arrow-text="<%=rb.getString("GaoJiChaXun")%>" style='margin-left:110px;'>
									<template slot="form">
										<div class='queryInfo'>
											<label><%=rb.getString("GuiZeMingCheng")%>:</label>
											<el-select v-model="params_status_form.ruleName">
												<el-option v-for="item in ruleNameOptions" :key="item.id" :label="item.text" :value="item.id">
												</el-option>
											</el-select>
										</div>
										<div class='queryInfo'>
											<label><%=rb.getString("XiaoZhanBianMa")%></label>
											<el-input v-model='params_status_form.serialNumber'></el-input>
										</div>
										<div class='queryInfo'>
											<label><%=rb.getString("KongZhiFangShi")%>:</label>
											<el-select v-model="params_status_form.controlType">
												<el-option v-for="item in controlTypeOptions" :key="item.id" :label="item.text" :value="item.id">
												</el-option>
											</el-select>
										</div>
										<div class='queryInfo'>
											<label><%=rb.getString("ZhuangTai")%>:</label>
											<el-select v-model="params_status_form.status">
												<el-option v-for="item in statusOptions" :key="item.id" :label="item.text" :value="item.id">
												</el-option>
											</el-select>
										</div>
									</template>
								</el-query> 
								<div style='position:absolute;right:0px;top:10px;'>
									<p class='suc_count'><span><i class='el-icon el-icon-status-accept'></i><%=rb.getString("JieShou")%></span><span>{{acceptNum}}</span></p>
									<p class='fail_count'><span><i class='el-icon el-icon-status-reject1'></i><%=rb.getString("JuJue")%></span><span>{{rejectNum}}</span></p>
								</div>
						</template>
			
						<!-- 主列表 -->
						<el-table-column label='' width="30" prop="" class-name="no-text-tips" v-if="omcOptBtnShow">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="300" prop="serialNumber"></el-table-column>
						<el-table-column label='<%=rb.getString("HostName")%>' width="250" prop="hostName"></el-table-column>		
						<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="200"  prop="status">
							<template slot-scope="scope">
								<div v-if="scope.row.status == 1">
									<span class='el-icon el-icon-status-accept' style='margin-right:5px;'></span><span><%=rb.getString("JieShou")%></span>
								</div>
								<div v-else>
									<span class='el-icon el-icon-status-reject1' style='margin-right:5px;'></span><span><%=rb.getString("JuJue")%></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("GuiZeMingCheng")%>' width="200" prop="tempName" ></el-table-column>
						<el-table-column label='<%=rb.getString("KongZhiFangShi")%>' width="150" prop="controlType"></el-table-column>
						<el-table-column label='<%=rb.getString("GuiHuaZhi")%>' width="180" prop="planningValue" :show-overflow-tooltip="false">
							<template slot-scope='scope'>
								<div v-if='scope.row.planningValue == null || scope.row.planningVlaue == ""'></div>
								<div v-else-if='scope.row.planningValue.split(";").length == 1'>{{scope.row.planningValue}}</div>
								<div v-else>
									<el-popover popper-class='infoTipPover' placement='bottom' trigger='hover' width='300'>
										<span v-html="scope.row.planningValue.replace(/;/g,'<br>')"></span>
										<span slot='reference'>{{scope.row.planningValue.split(';')[0]}}...</span>
									</el-popover>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("ShangBaoZhi")%>' width="180" prop="reportValue" :show-text-overflow-tips="false">
							<template slot-scope='scope'>
								<div v-if='scope.row.reportValue == null || scope.row.reportValue == ""'></div>
								<div v-else-if='scope.row.reportValue.split(";").length == 1'>{{scope.row.reportValue}}</div>
								<div v-else>
									<el-popover popper-class='infoTipPover' placement='bottom' trigger='hover' width='200'>
										<span v-html="scope.row.reportValue.replace(/;/g,'<br>')"></span>
										<span slot='reference'>{{scope.row.reportValue.split(';')[0]}}...</span>
									</el-popover>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("JiaoYanShiJian")%>' prop="accessTime"></el-table-column>
					</el-ctable>
					<el-cmenu ref="menu_status" :data="menus_status" @click="clickMenuStatus"></el-cmenu>
				</div>
			</div>
			<div class="egwAccessControlDiv" v-show="neType === 'EGW'">
				<div class="egwMainBox">
					<!-- 操作按钮 -->
					<div class="operationsBoxCls">
						<div class="newIconBoxCls-bt" style="right:0px;top:0px;" @click="goListPage" tip='List'>
							<span class="el-icon el-icon-circle-taskList"></span>
						</div>
					</div>
				
					<div class="egwDeviceTable">
						<el-ctable
							:url="egwDeviceTableUrl"
							:query-params="queryEgwParams" 
							ref="egwAccessControlTable" 
							id="egwAccessControlTable"
							:height="height" 
							@selection-change='deviceSelect'
							:page-size="pageSize" 
							:page-list="pageList" 
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div style="display: flex;align-items: center;">
									<h3 style="padding-left: 10px;">Access Status</h3>
									<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("eGWSheBeiBianMa")%> / ECI" style="margin-right: 30px;"></el-query>
									<div style="position:relative;">
										<el-switch v-model="autoEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#CFCFCF" :disabled="!egwOptBtnShow"></el-switch>
										<div class="maskBoxCls"  @click="autoEnableChange"></div>
									</div>
									
									<h3 style="padding-left: 10px;"></h3>
									<span class="enableExplain"><%=rb.getString("WeiShiBieECITiShi")%></span>
								</div>
								
							</template>
								<!-- 列表columns -->
							<el-table-column v-if="egwOptBtnShow" type="selection" width="45"></el-table-column>
							<el-table-column prop="" width="70" v-if="egwOptBtnShow">
								<template slot-scope="scope">
									<div style="display:flex;align-items: center;">
										<el-tooltip content='<%=rb.getString("JiaRuDaoBaiMingDan")%>' placement='bottom'>
											<span class="el-icon el-icon-operation-whitelist" @click="putInList(scope.row,'white','single')"></span>
										</el-tooltip>
										<el-tooltip content='<%=rb.getString("JiaRuDaoHeiMingDan")%>' placement='bottom'>
											<span class="el-icon el-icon-operation-blacklist" style="margin-left:10px;" @click="putInList(scope.row,'black','single')"></span>
										</el-tooltip>
									</div>
								</template>
							</el-table-column>
							<el-table-column label='ECI' min-width="120" prop="eci" show-overflow-tooltip></el-table-column>
							<el-table-column prop='status' label='<%=rb.getString("ZhuangTai")%>' min-width="120">
								<template slot-scope="scope">
									<div class="activeItemCls" v-if="scope.row.status == '0'">
										<span class="el-icon el-icon-circle-success"></span>
										<%=rb.getString("YiShiBie")%>
									</div>
									<div class="noActiveItemCls" v-if="scope.row.status == '1'">
										<span class="el-icon el-icon-circle-close"></span>
										<%=rb.getString("WeiShiBie")%>
									</div>
								</template>
							</el-table-column>
							<el-table-column prop='enb_serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
							<el-table-column prop='enb_name' label='eNB Name' min-width="150"></el-table-column>
							<el-table-column prop="egw_serial_number" label="<%=rb.getString("eGWSheBeiBianMa")%>" min-width="150"></el-table-column>
							<el-table-column prop="gw_ip" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
							<el-table-column prop="gw_port" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
							<el-table-column prop="event_time" label="<%=rb.getString("ShiJian")%>" min-width="150"></el-table-column>
						
						</el-ctable>
					</div>
				</div>
			</div>
		</div>

		<!--批量组件弹窗-->
		<el-bulk ref="viewListBulk" target="egwAccessControlTable" :list="selection" row-key="egwSn"  show-prop="egwSn"
			:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
			<template slot="button">
				<a class="linkbutton linkbutton_trend" @click="putInList('','white','batch')"><span><%=rb.getString("JiaRuBaiMingDan")%></span></a>
				<a class="linkbutton linkbutton_trend" @click="putInList('','black','batch')"><span><%=rb.getString("JiaRuHeiMingDan")%></span></a>
			</template>
		</el-bulk>	
		<!--egw slide -->
		<el-slide ref="egwListSlide" id="egwListSlide" :url='egwSlideUrl' :title="egwSlideTitle" :footer="egwSlideFooter" :header="egwSlideHeader" :position="egwSlidePosition"
			:height="egwSlideHeight"  :width='egwSlideWidth' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
		:height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
			
		</el-slide>
	</div>
</div>
<script>
	if(window.accessVue) {
		try {
			window.accessVue.$destroy();
		}catch(e){}
	}
	window.accessVue = new Vue({
		el:'#accessControlDiv',
		data(){
			return{
				omcAndEgwActiveName:'OMC',
				height:'100%',
				width:'100%',
			    statusUrl:'',
			    ruleUrl:'',
				params_accessStatus:{
					searchText:'',
					timeZone:timeZone,
					ruleName:'',
					serialNumber:'',
					controlType:'',
					status:''
				},
				params_status_form:{
					ruleName:'',
					serialNumber:'',
					controlType:'',
					status:''
				},
				params_accessRule:{
					searchText:'',
					timeZone:timeZone
				},
				params_rule_form:{searchText:''},
				ruleNameOptions:[],
				controlTypeOptions:[{id:'',text:'All'},{id:'TAC',text:'TAC'},{id:'IP',text:'IP'},{id:'ECGI',text:'ECGI'},{id:'GPS',text:'GPS'}],
				statusOptions:[{id:'',text:'All'},{id:0,text:'<%=rb.getString("JuJue")%>'},{id:1,text:'<%=rb.getString("JieShou")%>'}],
				menus_status:[],
				menus_rule:[],
				slideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				slideModal:'',
				operType:'',
				rowDataStatus:[],
				rowDataRule:[],
				acceptNum:0,
				rejectNum:0,

				egwDeviceTableUrl:"",
				selection:[],
				
				queryEgwParams:{
					search_text:'',
					like_fields:'egw_serial_number,eci',
					timeZone:timeZone,
				},
				egwSlideUrl:'',
				egwSlideTitle:'',
				egwSlideHeader:'',
				egwSlideFooter:'',
				egwSlidePosition:'',
				egwSlideHeight:'',
				egwSlideWidth:'',

				pageSize:100,
				pageList:[50,100,200,500],
				autoEnable:'false',
			}
		},
		computed: {
			omcOptBtnShow() {
				return writableMap['CODE_ADVANCE_ACCESS_CONTROL'] == true;
			},
			egwOptBtnShow() {
				return writableMap['CODE_EGW'] == true;
			},
			neType() {
				var vm = this;
				return sysMain.headType.toUpperCase();
			},
		},
		watch: {
			neType(types) {
				if(types == 'EGW') {
					this.getAutoSaveBlackSwitch();
				}else{
					this.getRuleNameOptions();
				}
			}
		},
		methods:{
			init(){
				var vm = this;
                if(writableMap['CODE_ADVANCE_ACCESS_CONTROL'] != undefined){
                    vm.statusUrl = '${ctx}/son/access/queryAccessStatusInfoPageList.action';
			        vm.ruleUrl = '${ctx}/son/access/queryAccessTempInfoPageList.action';
                    vm.getRuleNameOptions();
                }
                if(writableMap['CODE_EGW'] != undefined){
                    vm.egwDeviceTableUrl = "${ctx}/egw/enbCheck/getEgwEnbCheckList.action";
                    vm.getAutoSaveBlackSwitch();
                }
			},
            // 获取接入规则名称下拉选项
            getRuleNameOptions(){
                axios.post("${ctx}/son/access/queryAccessTempByOperCodeSelect.action").then(function(response){
                    var data = response.data;
                    vm.ruleNameOptions = data;
                })
            },
			/**
			 * 接入状态->状态列格式化
			 * @param row {object} 行字段
			   @param column {object} 列属性
			   @param value {string} 当前字段值
			   @param index {number} 索引
			*/
			accessStatusFmt(row,column,value,index){
				if(value == "1"){
					return '<%=rb.getString("JieShou")%>';
				}else{
					return '<%=rb.getString("JuJue")%>';
				}
			},
			//点击页面其他地方关闭菜单
			handerClose(){
				 this.$refs.menu_status.hide();
				 this.$refs.menu_rule.hide();
			},
			/**
			 * 接入状态菜单点击方法
			 * @param row {object} 点击行数据
			   @param ev {object} 鼠标事件
			*/
			optClick(row,ev){
				var vm = this;
				vm.rowDataStatus = row;
				var blackFlag = false;
				var whiteFlag = false;
				//black为true说明此基站已经在黑名单中
				if(row.black == "true"){
					blackFlag = true
				}
				//white为true说明此基站已经在白名单中
				if(row.white == "true"){
					whiteFlag = true;
				}
				vm.menus_status= [
			          {label:'<%=rb.getString("JiaRuHeiMingDan")%>',cls:"el-icon el-icon-operation-blacklist",code:'black',disable:blackFlag},
			          {label:'<%=rb.getString("JiaRuBaiMingDan")%>',cls:"el-icon el-icon-operation-whitelist",code:'white',disable:whiteFlag},
			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del'}
			    ]
		    	vm.$nextTick(function(){
		    		document.body.click();
  		    		vm.$refs.menu_status.show(ev);
		    	});
			},
			/**
			 * 点击接入状态下拉菜单项触发方法
			 * @param ev {object} 鼠标事件
			*/
			clickMenuStatus(ev){
				var codes = {
	    	    	black:this.addblacklist,
	    	    	white:this.addwhitelist,
	    	    	del:this.delStatus
	    	    }
    	    	if(codes[ev.code]){
    	    		codes[ev.code]()
    	    	}
			},
			/**
			 * 接入状态列表模糊查询
			 * @param val {string} 查询值
			*/
			query(val){
				this.resetQuery();
				this.params_accessStatus.searchText  = val;
			},
			//高级查询
			advanceQuery(){
				this.params_accessStatus.searchText = '';
				Object.assign(this.params_accessStatus,this.params_status_form);
			},
			//重置查询
			resetQuery(){
				this.params_status_form.ruleName = '';
				this.params_status_form.serialNumber = '';
				this.params_status_form.controlType = '';
				this.params_status_form.status = '';
			},
			//接入规则模糊查询
			queryRule(){
				Object.assign(this.params_accessRule,this.params_rule_form);
			},
			//加入黑名单方法
			addblacklist(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingJiaRuHeiMingDan")%>';
				var params = {
						serialNumber:vm.rowDataStatus.serialNumber
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/son/access/addAccessBlackList.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctableAccessStatus.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			//加入白名单方法
			addwhitelist(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingJiaRuBaiMingDan")%>';
				var params = {
						serialNumber:vm.rowDataStatus.serialNumber
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/son/access/addAccessWhiteList.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctableAccessStatus.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			//删除设备
			delStatus(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				var params = {
						id:vm.rowDataStatus.id
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/son/access/delAccessStatusInfosById.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctableAccessStatus.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			//接入规则菜单点击方法
			optClickRule(row,ev){
				var vm = this,
					status = row.status,
					statusText = '',
					statusIcon = '',
					operFlag = status == '0' ? false : true;
				vm.rowDataRule = row;
				vm.menus_rule= [
			          {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
			          {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_ADVANCE_ACCESS_CONTROL hidden",code:'edit',disable:operFlag},
			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ADVANCE_ACCESS_CONTROL hidden",code:'del',disable:operFlag},
			          {label:'<%=rb.getString("JieRuXiangQing")%>',cls:"el-icon el-icon-operation-accesscontrol",code:'detail'}
			    ]
		    	vm.$nextTick(function(){
		    		document.body.click();
  		    		vm.$refs.menu_rule.show(ev);
		    	});
			},
			//点击接入规则下拉菜单项触发方法
			clickMenuRule(ev){
				var codes = {
					active:this.activeTemp,
					del:this.delTemp,
					edit:this.editTemp,
					info:this.viewTemp,
					detail:this.goDetail
				}
				if(codes[ev.code]){
					codes[ev.code]()
				}
			},
			//添加规则
			addRule(){
				var vm = this;
				vm.slideUrl = '${ctx}/son/access/goAccessControlAddRule.action',
				vm.slideTitle = '<%=rb.getString("XinZengGuiZe")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operType = 'add';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			//跳转黑白名单List页面
			goList(){
				var vm = this;
				vm.slideUrl = '${ctx}/son/access/goAccessControlList.action',
				vm.slideFooter = false;
				vm.slideHeader = false;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operType = 'view';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			//划出框取消方法
			cancelSlide(){
				if(this.operType == "view"){
					this.$refs.slide.hide();
				}else{
					eventBus.$emit("cancel-rule");
				}
			},
			//划出框确定方法
			saveSlide(){
				eventBus.$emit('save-rule');
			},
			//启用/禁用模板
			activeTemp(){
				var vm = this;
				var params = {
						id : vm.rowDataRule.id
				}
				if(vm.rowDataRule.status == "0"){
					params.status = "1"
				}else{
					params.status = "0"
				}
				axios.post("${ctx}/son/access/updateAccessTempEnable.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ctableAccessRule.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			//删除模板
			delTemp(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChuGuiZe")%>';
				var params = {
						id:vm.rowDataRule.id
				}
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/son/access/delAccessTempById.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.ctableAccessRule.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
			//修改模板
			editTemp(){
				var vm = this;
				vm.slideUrl = '${ctx}/son/access/goAccessControlAddRule.action',
				vm.slideTitle = '<%=rb.getString("XiuGaiGuiZe")%>';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operType = 'edit';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-info");
	    	    });
			},
			//查看模板
			viewTemp(){
				var vm = this;
				vm.slideUrl = '${ctx}/son/access/goAccessControlAddRule.action',
				vm.slideTitle = '<%=rb.getString("ChaKanGuiZe")%>';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operType = 'view';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-info");
	    	    });
			},
			//查看接入情况
			goDetail(){
				var vm = this;
				vm.slideUrl = '${ctx}/son/access/goAccessControlDetail.action',
				vm.slideFooter = false;
				vm.slideHeader = false;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operType = 'view';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			changeStatus(newVal,id){
				var vm = this;
				var params = {
						id : id,
						status : newVal
				}
				axios.post("${ctx}/son/access/updateAccessTempEnable.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.ctableAccessRule.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			loadSuccessStatus(data){
				this.rejectNum = data.properties[0];
				this.acceptNum = data.properties[1];
			},
			//egw 请求开关状态
			getAutoSaveBlackSwitch(){
				var vm = this;
				axios.post('${ctx}/egw/enbCheck/getAutoSaveBlackSwitch.action').then(function(response){
					var data = response.data;
					vm.autoEnable = data ? 'true' : 'false';
				})
			},
			// 前往egw 黑白名单
			goListPage(){
				var vm = this;
				vm.egwSlideHeader = false;
				vm.egwSlideUrl = '${ctx}/egw/pageForward/toEgwEnbWhiteBlackListPage.action';
				vm.egwSlideFooter = false;
				vm.egwSlidePosition = 'top';
				vm.egwSlideHeight = '100%';
				vm.egwSlideWidth = '100%';
				vm.egwSlideTitle = '';
				
				vm.$refs.egwListSlide.showSlide(()=>{})
			},
			//egw 设备表格选择事件
			deviceSelect(selection){
				var vm = this;
				vm.selection = selection;
			},
			// egw设备表格 模糊查询
			queryDevice(val){
				var vm = this;
				vm.queryEgwParams.search_text= val;
			},
			// 自动加入黑名单
			autoEnableChange(){
				var vm = this,
					url = '${ctx}/egw/enbCheck/updateAutoSaveBlackSwitch.action',
					params = {
						switch:vm.autoEnable == 'true'? 'false' : 'true'
					};
				axios.post(url,stringify(params)).then(function(response){
					var data = response.data;
					var message = '<%=rb.getString("ChengGong")%>';
					if(data["success"]){
						vm.$message({
							message:message,
							type:'success',
						})
						vm.autoEnable = vm.autoEnable == 'true'? 'false' : 'true';
					}else{
						vm.$message.error(data["message"])
					}
				})
				event.stopPropagation();
			},
			/**
			* 加入黑名单或白名单
			* @param row: 行数据
			* @param listType: 加入名单类型  white：白 black：黑
			* @param opType: 单个/批量  single/batch
			*/
			// 加入黑名单或白名单
			putInList(row,listType,opType){
				var vm = this,
					url = "",
					dataList=[];
				if(opType == "single"){
					dataList.push({eci:row.eci,egw_serial_number:row.egw_serial_number})
				}else{
					vm.selection.map((item)=>{
						dataList.push({eci:item.eci,egw_serial_number:item.egw_serial_number})
					})
				}
				if(listType == "black"){
					var confirmStr = '<%=rb.getString("QueRenJiaRuHeiMingDan")%>';
					url = "${ctx}/egw/enbCheck/saveECIToBlackList.action";
				}else{
					var confirmStr = '<%=rb.getString("QueRenJiaRuBaiMingDan")%>';
					url = "${ctx}/egw/enbCheck/saveECIToWhiteList.action";
				}
				let paramsData = JSON.stringify(dataList);
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url,paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
								message:message,
								type:'success',
							})
							vm.$refs.egwAccessControlTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {
					
				})
			},
		},
		mounted(){
			this.init();
		}
	})
</script>
