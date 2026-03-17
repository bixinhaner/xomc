<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	
	
	#PCIConfusedPage .pageBody {
		position: relative;
		overflow: auto;
	}
	.PciConfusedSolve .el-icon:before{
		color:#67D972;
	}
	.PciConfusedNoSolve .el-icon:before{
		color:#E88282;
	}
	#suggestPciForm .el-input{
		padding-top: 7px;
	}
	.timerSty{
		font-size: 16px;
		font-weight: 600;
		margin-left: 10px;
	}
	.controlledSuggestDialog{
		display: flex;
		padding-left: 20px;
	}
	.controlledSuggestDialog>div{
		height: 26px;
		line-height: 26px;
	}
	.controlledSuggestDialog > div:first-child{
		margin-right: 40px;
	}
	.controlledSuggestDialog > div:nth-child(2){
		margin-right: 20px;
		font-size: 16px;
		font-weight: 600;
	}
	.controlledSuggestDialog > div:nth-child(3){
		width: 95px;
		border: 1px solid #E9E9E9;
		text-align: center;
		cursor: pointer;
		font-size: 12px;
		border-radius: 2px;
	}
	.controlledSuggestDialog > div:nth-child(3):hover{
		border: 1px solid #4D84FF;
	}
	.recalculatePrompt{
		padding-top: 20px;
		margin-left: 143px;
		font-size: 12px;
	}
	#suggestPciForm{
		padding-bottom: 10px;
	}
	#recalculateDialog .el-dialog__body{
		padding: 0px!important;
	}
	#recalculateDialog .el-dialog__body .el-form-item{
		margin-bottom: 10px!important;
	}
	.freeSuggestDialog{
		display: flex;
		padding-left: 20px;
		margin-bottom: 20px;
	}
	.freeSuggestDialog>div{
		height: 26px;
		display: flex;
		align-items: center;
	}
	.freeSuggestDialog > div:first-child{
		margin-right: 20px;
		width: 100px;
	}
	.freeSuggestDialog > div:nth-child(2){
		margin-right: 20px;
		font-size: 16px;
		font-weight: 600;
	}
	.freeSuggestDialog > div:nth-child(3){
		cursor: pointer;
		font-size: 16px;
		font-weight: 900;
		margin-right: 20px;
	}
	.freeSuggestDialog > div:nth-child(4){
		display: flex;
		align-items: center;
	}
	.freeSuggestDialog .el-icon:before{
		font-size: 20px!important;
	}
	#recalculateDialog .el-form-item__error{
		white-space: nowrap!important;
		padding-top: 0px!important;
	}
	.issueType{
		display: flex;
		padding-left: 20px;
	}
	.issueType>div{
		display: flex;
		align-items: center;
	}
	.issueType>div:first-child{
		width: 100px;
	}
	.startIconNoClick .el-icon:before{
		color:#C9DAFF!important;
	}
	.queryInfo{
		display:inline-block;
		margin-right:35px;
	}
	.queryInfo label{
		display:block;
		line-height:24px;
	}
	
</style>
<div class="pageDefault" id='PCIConfusedPage' style='border: 0;'>
	<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:20px;top:12px;" @click="PCIConfusedSetting" tip="<%=rb.getString("SheZhi")%>">		
		<span class="el-icon el-icon-circle-setting"></span>
	</div>
	<template>
		<div class="pageBody" style='border: 0;'> <!-- :data="data" :url="queryPCIConfusedUrl"   -->
			<el-ctable :url="queryPCIConfusedUrl" :time="6" :query-params="queryParams" id="PCIConfusedTable" ref="PCIConfusedTable" :height="height" :page-size="pageSize" pagination="true">
				<template slot="toolbar">
					<div>
						<el-query @query="confusedQuery" @advance-query="confusedQueryAdvance" @reset="confusedQueryReset" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>"
							placeholder="<%=rb.getString("HostName")%>/Cell ECGI/Cell PCI">
							<template slot="form">
								<div class='queryInfo'>
									<label><%=rb.getString("HostName")%></label>
									<el-input v-model="params_advance.cellName" size="mini" style='width:200px'></el-input>
								</div>
								<div class='queryInfo'>
									<label>Cell ECGI</label>
									<el-input v-model="params_advance.cellEcgi" size="mini" style='width:200px'></el-input>
								</div>
								<div class='queryInfo'>
									<label>Cell Earfcn</label>
									<el-input v-model="params_advance.cellEarfcn" size="mini" style='width:200px'></el-input>
								</div>
								<div class='queryInfo'>
									<label>Cell PCI</label>
									<el-input v-model="params_advance.cellPci" size="mini" style='width:200px'></el-input>
								</div>
								<div class='queryInfo'>
									<label>Type</label>
									<el-select filterable v-model="params_advance.type">
										<el-option v-for="item in PciTypeList" :label="item.name" :value="item.value"></el-option>
									</el-select>
								</div>
								<div class='queryInfo'>
									<label>Suggest PCI</label>
									<el-input v-model="params_advance.suggestPci" size="mini" style='width:200px'></el-input>
								</div>
								<div class='queryInfo'>
									<label>Neighbor PCI</label>
									<el-input v-model="params_advance.neighborPci" size="mini" style='width:200px'></el-input>
								</div>
								<div class='queryInfo'>
									<label><%=rb.getString("ZhuangTai")%></label>
									<el-select filterable v-model="params_advance.status">
										<el-option v-for="item in statusList" :label="item.name" :value="item.value"></el-option>
									</el-select>
								</div>
							</template>
						</el-query>
					</div>
				</template>
				<el-table-column prop="operation" label="" width="30">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				
				<el-table-column prop="cellName" label="<%=rb.getString("HostName")%>"></el-table-column>
				<el-table-column prop="cellEcgi" label="Cell ECGI"></el-table-column>
				<el-table-column prop="cellEarfcn" label="Cell Earfcn"></el-table-column>
				<el-table-column prop="cellPci" label="Cell PCI" ></el-table-column>
				<el-table-column prop="type" label="Type" >
					<template slot-scope="scope">
						<div  v-if="scope.row.type == 0">
							<span><%=rb.getString("PCIChongTu")%></span>
						</div>
						<div  v-if="scope.row.type == 1">
							<span><%=rb.getString("PCIHunXiao")%></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop="suggestPci" label="Suggest PCI" ></el-table-column>
				<el-table-column prop="neighborCell" label="Neighbor Cell" ></el-table-column>
				<el-table-column prop="status" label="<%=rb.getString("ZhuangTai")%>" width="180" sortable>
					<template slot-scope="scope">
						<div class="PciConfusedSolve" v-if="scope.row.status == 1">
							<span class="el-icon el-icon-status-success" style='margin-right:5px;'></span><span><%=rb.getString("YiJieJue")%></span>
						</div>
						<div class="PciConfusedNoSolve" v-if="scope.row.status == 0">
							<span class="el-icon el-icon-status-failed"></span><span style='margin-left:5px;'><%=rb.getString("WeiJieJue")%></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop="eventTime" label="<%=rb.getString("GuZhangShiJian")%>" sortable></el-table-column>
				<el-table-column prop="updateTime" label="<%=rb.getString("GengXinShiJian")%>" ></el-table-column>
				<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" ></el-table-column>
				<el-table-column prop="count" label="<%=rb.getString("TiaoShu")%>" ></el-table-column>
			</el-ctable>
			<el-cmenu ref="menu" @click="clickMenu" :data="menus"></el-cmenu>
		</div>
	</template>
	<!--过滤冲突 slide -->
	<el-slide  ref="PCIConfusedSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" id="PCIConfusedSlide"
		:height="slideHeight" :modal='modal' :width='slideWidth' @ok="slideOk"  @cancel="slideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!--PCI建议 弹窗 -->
	<el-dialog 
		title='Suggest PCI' 
		:visible.sync="showSuggestDialog" top="30vh" 
		ref="suggestDialog" 
		:width="suggestDialogWidth" 
		:close-on-click-modal="false"  
		@close='closeSuggestDialog'  
		@success="openDialogSuc">
		
		<div class="controlledSuggestDialog" v-if="controlledType == '1'">
			<div>Suggest PCI</div>
			<div>{{suggestPciValue}}</div>
			<div @click="recalculateClick"><span class="el-icon el-icon-common-refresh" style="margin-right:5px;"></span><%=rb.getString("ChongXinJiSuan")%></div>
		</div>
		<div class="recalculatePrompt" >
			<span v-show="recalculatePciHintShow"><%=rb.getString("ChongXinJiSuanDengDaiTiShi")%></span>
		</div>
		<div class="freeSuggestDialog" v-if="false">
			<div>Suggest PCI</div>
			<div>185</div>
			<div v-if="issueStatus !== 'suspend'">{{minute}}:{{second}}</div>
			<div @click="suggestPciPause" v-if="issueStatus == 'wait'"><span class="el-icon el-icon-circle-log" style="margin-right:5px;"></span></div>
			<div @click="suggestPciStart" v-if="issueStatus == 'suspend'"><span class="el-icon el-icon-circle-exe" style="margin-right:5px;"></span></div>
			<div v-if="issueStatus == 'inProgress'" class="startIconNoClick"><span class="el-icon el-icon-circle-log" style="margin-right:5px;"></span></div>
		</div>
		<div class="issueType" v-if="false">
			<div style="margin-right:20px">下发状态</div>
			<div v-if="issueStatus == 'wait'"><span class="el-icon el-icon-status-waiting1" style="margin-right:10px;"></span>等待下发</div>
			<div v-if="issueStatus == 'suspend'"><span class="el-icon el-icon-status-suspend" style="margin-right:10px;"></span>已暂停</div>
			<div v-if="issueStatus == 'inProgress'"><span class="el-icon el-icon-status-inProgress" style="margin-right:10px;"></span>下发中</div>
		</div>
		<el-dialog title='Suggest PCI' id="recalculateDialog" :visible.sync="showRecalculateDialog" top="30vh" ref="suggestDialog" :width="suggestDialogWidth" 
			:close-on-click-modal="false"  @close='closeRecalculatePci' append-to-body>
			<el-form  :model="suggestPciForm" ref="suggestPciForm" :rules="suggestRules" label-position="left" id="suggestPciForm">
				<el-form-item  style="margin-left:40px;margin-top:10px;" :label="rangeLabel" prop='rangeStart' class="rangeClass" label-width="150px">
					<el-input v-model.trim='suggestPciForm.rangeStart' oninput="if(value != ''){value= Number(value.replace(/[^\d]/g,''))}" maxlength="4" style="width:60px;"></el-input>
					<span style="margin:0px 5px;">-</span>
					<el-input v-model.trim='suggestPciForm.rangeEnd' oninput="if(value != ''){value= Number(value.replace(/[^\d]/g,''))}" maxlength="4" style="width:60px;"></el-input>
				</el-form-item>
				<el-form-item prop='reuseDistance' style="margin-left:40px;" label="<%=rb.getString("ZuiXiaoFuYongJuLi") %>" :label-width="reuseDistanceLableWidth">
					<el-input v-model.trim='suggestPciForm.reuseDistance' style="width:60px;" oninput="if(value != ''){value= Number(value.replace(/[^\d]/g,''))}" maxlength="4"></el-input>
				</el-form-item>
			</el-form>
			<span slot="footer">
				<div>
					<el-button type="primary" @click="recalculatePciSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeRecalculatePci"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</span>
		</el-dialog>
		<span slot="footer" v-if="controlledType !== '2'">
			<div>
				<el-button type="primary" @click="SuggestPciSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeSuggestDialog"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
</div>
<script type="text/javascript">
new Vue({
	el:'#PCIConfusedPage',
	data(){
		var vm = this;
		var validateRangeStart = (rule,value,callback) => {
			var rangeEnd = vm.suggestPciForm.rangeEnd,
				start = parseInt(vm.rangeSettingStart),end = parseInt(vm.rangeSettingEnd),
				errorText = '<%=rb.getString("PCIFanWei")%>' + '(' + start + '-' + end + ')'
			if(value && rangeEnd){
				if(parseInt(value,10) < start || parseInt(rangeEnd,10) > end){
					callback(new Error(errorText))
				}else if(parseInt(value,10) >= parseInt(rangeEnd,10)){
					callback(new Error(errorText))
				}else{
					callback();
				}
			}else{
				callback(new Error('<%=rb.getString("FanWeiBuNengWeiKong")%>'))
			}
		};
		var validateReuseDistance = (rule,value,callback) => {
			var maxReuseDistance = parseInt(vm.minMultiplexingDistance),
				errorText = '<%=rb.getString("FanWei")%>' + '(1-' + maxReuseDistance  + ')';
			if(value){
				if(value > maxReuseDistance || value<1){
					callback(new Error(errorText))
				}else{
					callback()
				}
			}else{
				callback(new Error('<%=rb.getString("FanWeiBuNengWeiKong")%>'))
			}
		};
		return {
			queryParams:{  // 查询
				searchText:'', // 搜索 value
				timeZone:timeZone,
				cellName:'',
				cellEcgi:'',
				cellEarfcn:'',
				cellPci:'',
				type:'',
				suggestPci:'',
				neighborCell:'',
				status:'',
				likeFields:'cellName,cellEcgi,cellPci'
			},
			params_advance:{      // 高级查询表单参数
				cellName:'',
				cellEcgi:'',
				cellEarfcn:'',
				cellPci:'',
				type:'',
				suggestPci:'',
				neighborPci:'',
				status:'',
			},
			PciTypeList:[
				{name:'All',value:''},
				{name:'<%=rb.getString("PCIChongTu")%>',value:'0'},
				{name:'<%=rb.getString("PCIHunXiao")%>',value:'1'},
			],
			statusList:[
				{name:'All',value:''},
				{name:'<%=rb.getString("WeiJieJue")%>',value:'0'},
				{name:'<%=rb.getString("YiJieJue")%>',value:'1'},
			],
			time:'',
			minutes:1,
			seconds:12,
			rowData:[],  
			menus:[],
			slideUrl:'',
		    slideTitle:'',
		    slideHeader:'',
		    slideFooter:'',
		    slidePosition:'',
		    slideHeight:'',
		    slideWidth:'',
		    modal:false,
			queryPCIConfusedUrl:'${ctx}/pci/queryConflictConfusionList.action',
			height:'100%',
			pageSize:50,
			loading:false,
			controlledType:'1',
			detectorSwitch:'1',
			optimizeSwitch:'1',
			showSuggestDialog:false,
			showRecalculateDialog:false,
			recalculatePciHintShow:false,
			issueStatus:'wait',
			suggestDialogWidth:'460px',
			slideType:'',
			suggestPciValue:'',
			rangeSettingStart:'',
			rangeSettingEnd:'',
			minMultiplexingDistance:'',

			suggestPciForm:{
				rangeStart:'',
				rangeEnd:'',
				reuseDistance:'',
			},
			suggestRules:{
				rangeStart:[
					{validator:validateRangeStart,trigger:'blur'}
				],
				reuseDistance:[
					{validator:validateReuseDistance,trigger:'blur'}
				],
			},
			data:[
				{
					id:1,
					cellName:'zhang',
					cellEcgi:'32590',
					cellEarfcn:'80',
					cellPci:'55',
					type:'PCI冲突',
					suggestPci:'12',
					neighborCell:'Cell1 Name=name1,cell1-ECGI=254,CELL1-PCI=21',
					status:'1',
					eventTime:'2020-12-04 10:44:00',
					updateTime:'2020-12-04 11:55:00',
					count:'12'
				},
				{
					id:2,
					cellName:'xin',
					cellEcgi:'66655',
					cellEarfcn:'10',
					cellPci:'33',
					type:'PCI冲突',
					suggestPci:'12',
					neighborCell:'Cell1 Name=name1,cell1-ECGI=254,CELL1-PCI=21',
					status:'2',
					eventTime:'2020-12-04 10:44:00',
					updateTime:'2020-12-04 11:55:00',
					count:'3'
				},
				{
					id:3,
					cellName:'a',
					cellEcgi:'75435',
					cellEarfcn:'51',
					cellPci:'13',
					type:'PCI冲突',
					suggestPci:'36',
					neighborCell:'Cell1 Name=name1,cell1-ECGI=254,CELL1-PCI=21',
					status:'2',
					eventTime:'2020-12-04 10:44:00',
					updateTime:'2020-12-04 11:55:00',
					count:'4'
				}
			]
		}
	},
	watch:{
		second:{
			handler(newVal){
				this.num(newVal);
			}
		},
		minute:{
			handler(newVal){
				this.num(newVal);
			}
		}
	},
	computed:{
		second:function(){
			return this.num(this.seconds);
		},
		minute:function(){
			return this.num(this.minutes);
		},
		rangeLabel:function(){
			return '<%=rb.getString("PCIFanWei")%>' + '(' + this.rangeSettingStart + '-' + this.rangeSettingEnd +')';
		},
		reuseDistanceLableWidth:function(){
			return language == 'en'? '280px' : '150px';
		},
		optBtnShow() {
			return writableMap['CODE_ADVANCE_SON'] == true;
		},
	},
	methods:{
		// 初始化
		init(){
			var vm = this;
			axios.post("${ctx}/pci/getSettings.action").then((res) => {
				var data = res.data;
				vm.detectorSwitch = data.pciCheckSwitch;
				vm.optimizeSwitch = data.pciOptimizeSwitch;
				vm.controlledType = data.pciControlSwitch;
				vm.minMultiplexingDistance = data.minMultiplexingDistance;
				if(data.pciRange){
					vm.rangeSettingStart = data.pciRange.split('-')[0];
					vm.rangeSettingEnd = data.pciRange.split('-')[1];
				}
			});
		},
		// 搜索点击事件
		confusedQuery(val){ 
			var vm = this;
			vm.queryParams.searchText = val;
			vm.queryParams.cellName = vm.params_advance.cellName = "";
			vm.queryParams.cellEcgi = vm.params_advance.cellEcgi = "";
			vm.queryParams.cellEarfcn = vm.params_advance.cellEarfcn = "";
			vm.queryParams.cellPci = vm.params_advance.cellPci = "";
			vm.queryParams.type = vm.params_advance.type = "";
			vm.queryParams.suggestPci = vm.params_advance.suggestPci = "";
			vm.queryParams.neighborCell = vm.params_advance.neighborPci = "";
			vm.queryParams.status = vm.params_advance.status = "";
			vm.$refs.PCIConfusedTable.reset()
		},
		// 表格高级查询
		confusedQueryAdvance(){
			var vm = this;
			vm.queryParams.searchText = '';
			vm.queryParams.cellName = vm.params_advance.cellName ;
			vm.queryParams.cellEcgi = vm.params_advance.cellEcgi ;
			vm.queryParams.cellEarfcn = vm.params_advance.cellEarfcn ;
			vm.queryParams.cellPci = vm.params_advance.cellPci ;
			vm.queryParams.type = vm.params_advance.type ;
			vm.queryParams.suggestPci = vm.params_advance.suggestPci ;
			vm.queryParams.neighborCell = vm.params_advance.neighborPci ;
			vm.queryParams.status = vm.params_advance.status ;
			vm.$refs.PCIConfusedTable.reset()
		},
		// 表格高级查询重置
		confusedQueryReset(){
			var vm = this;
			vm.params_advance.cellName = "";
			vm.params_advance.cellEcgi = "";
			vm.params_advance.cellEarfcn = "";
			vm.params_advance.cellPci = "";
			vm.params_advance.type = "";
			vm.params_advance.suggestPci = "";
			vm.params_advance.neighborPci = "";
			vm.params_advance.status = "";
		},
		// 点击页面其他地方菜单收起
	    handerClose(){
	        this.$refs.menu.hide();
	    },
		// 倒计时转换
		num(n){
			return n < 10 ? '0' + n : '' + n
		},
		countDown(){
			var vm = this;
			vm.time = window.setInterval(function(){
				if(vm.seconds === 0 && vm.minutes !== 0){
					vm.seconds = 59 ;
					vm.minutes -= 1 ;
				}else if(vm.minutes === 0 &&vm.seconds === 0){
					vm.seconds = 0 ;
					vm.issueStatus = 'inProgress';
					window.clearInterval(time);
				}else{
					vm.seconds -= 1 ;
				}
			},1000)
		},
		/**
		* 点击更多操作出现菜单
		* @param row{object}   行数据
		* @param ev{object}   event数据
		*/ 
	    optClick(row,ev){
	    	//在启用状态下  禁止修改 删除
			var vm = this,suggestFlag;

	    	vm.rowData = row;
			vm.suggestPciValue = row.suggestPci;
	    	var rowStatus = row.status;
			if(vm.controlledType == '1' && rowStatus == '0'){
				suggestFlag = false;
			}else{
				suggestFlag = true;
			}
	    	this.menus = [
	    		{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'},
	    		{label:'PCI Suggest',cls:"el-icon el-icon-operation-change ",disable:suggestFlag,code:'suggest'},
	    	]
	    	this.$nextTick(function(){
	    		document.body.click();
		    	vm.$refs.menu.show(ev);
	    	})
	    },
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
	    clickMenu(ev){
	    	var codes = {
	    		view:this.viewPCIConfused,	// 查看
	    		suggest:this.suggestPCI,	// 建议PCI
	    	}
			codes[ev.code](this.rowData["id"]);
	    },
		/**
		* 查看
		* @param id{number}  id
		*/ 
	    viewPCIConfused(id){
	    	var vm = this;
			vm.slideType = 'view'
			anrSettingVue.pciSlideUrl = "${ctx}/pci/goConflictConfusionDetail.action";
			anrSettingVue.pciSlideHeight = '100%';
			anrSettingVue.pciSlideWidth = '100%';
			anrSettingVue.pciSlideFooter = false;
			anrSettingVue.pciSlidePosition = 'top';
			anrSettingVue.pciSlideHeader = true;
			anrSettingVue.pciSlideTitle = '<%=rb.getString("XiangQing")%>';
			anrSettingVue.$refs.PCIConfusedSlide.showSlide(function(){
				anrSettingVue.modal = false;
				eventBus.$emit('detail-info',id);
			});
	    },
		/**
		* PCI 建议 
		* @param id{number}  id
		*/ 
	    suggestPCI(id){
	    	var vm = this;
			vm.showSuggestDialog = true;
			// vm.countDown();
	    },
		//点击设置按钮 进入setting页面
	    PCIConfusedSetting(){
			var vm = this;
			vm.slideType = 'set';
			anrSettingVue.pciSlideUrl = "${ctx}/pci/goConflictConfusionSetting.action";
			anrSettingVue.pciSlideHeight = '100%';
			anrSettingVue.pciSlideWidth = '100%';
			anrSettingVue.pciSlideFooter = true;
			anrSettingVue.pciSlidePosition = 'top';
			anrSettingVue.pciSlideHeader = true;
			anrSettingVue.pciSlideTitle = '<%=rb.getString("SheZhi")%>';
			anrSettingVue.$refs.PCIConfusedSlide.showSlide(function(){
				anrSettingVue.modal = false;
			});
	    },
		// 过滤设置确认
		slideOk(){
			var vm = this;
			eventBus.$emit('confusedSetting-ok')
		},
		// 详情页面关闭事件
		slideCancel(){
			var vm = this;
			if(vm.slideType == 'set'){
				eventBus.$emit('confusedSetting-cancel');
			}else{
				anrSettingVue.$refs.PCIConfusedSlide.hide();
			}
			
		},
		// slide页面关闭事件
		slideClose(){
			var vm = this;
			anrSettingVue.$refs.PCIConfusedSlide.hide();
			vm.$refs.PCIConfusedTable.refresh();
			vm.init();
		},
		// 开始
		suggestPciStart(){
			var vm = this;
			vm.issueStatus = 'wait';
			vm.countDown();
		},
		// 暂停
		suggestPciPause(){
			var vm = this;
			vm.issueStatus = 'suspend';
			window.clearInterval(vm.time);
		},
		// 建议PCI弹窗关闭
		closeSuggestDialog(){
			var vm = this;
			vm.showSuggestDialog = false;
			
		},
		// 重新计算PCI弹窗关闭
		closeRecalculatePci(){
			var vm = this;
			vm.showRecalculateDialog = false;
			vm.suggestPciForm.rangeStart = '';
			vm.suggestPciForm.rangeEnd = '';
			vm.suggestPciForm.reuseDistance = '';
		},
		// 打开弹窗成功回调
		openDialogSuc(){
			var vm = this;
		},
		// 优化开关改变事件
		optimizeSwitchChange(val){
			var vm = this;
			if(val !== '1'){
				vm.controlledType = '2';
			}
		},
		// 重新计算点击事件 打开重新计算窗口
		recalculateClick(){
			var vm = this;
			vm.showRecalculateDialog = true;
		},
		// 重新计算PCI值 确定
		recalculatePciSubmit(){
			var vm = this;
			vm.$refs.suggestPciForm.validate((valid) => {
				if(valid){
					var params = {
						id:vm.rowData.id,
						pciRange:vm.suggestPciForm.rangeStart + '-' + vm.suggestPciForm.rangeEnd,
						minMultiplexingDistance:vm.suggestPciForm.reuseDistance,
						suggestPci:vm.rowData.suggestPci
					}
					vm.showRecalculateDialog = false;
					vm.recalculatePciHintShow = true;
					axios.post("${ctx}/pci/calculatePCI.action",stringify(params)).then(function(response){
						vm.suggestPciValue = response.data;
						vm.recalculatePciHintShow = false;
		 			})
				}else{
					return false;
				}
			})
		},
		// 建议PCI值确定提交
		SuggestPciSubmit(){
			var vm = this,params;
			params = {
				id:vm.rowData.id,
				suggestPci:vm.suggestPciValue
			}
			axios.post("${ctx}/pci/doExecutePCISuggest.action",stringify(params)).then(function(response){
				if(response.data.success){
					vm.$message({
							message:'<%=rb.getString("ChengGong")%>',
							type:'success',
						})
					vm.showSuggestDialog = false;
					vm.$refs.PCIConfusedTable.refresh();
				}else{
					vm.$message.error(response.data["message"])
				}
				
			})
		}

	},
	mounted(){
		this.init();
		eventBus.$on('hide-PCIConfusedSlide',this.slideClose);
	}
	
})

</script> 