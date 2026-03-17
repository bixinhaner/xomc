<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>

<style type="text/css">
#anrSettingPage .h100{
	height: 100%
}
#anrSettingPage .settingsBtn{
    top:45px;
	z-index:10;
	right:20px;
}

#anrSettingPage .errorColor:before{
	font-size:18px;
}
#anrSettingPage .successColor:before{
	color:#67D972;
	font-size:18px;
}
#anrSettingPage .resultTip{
	display:flex;
}
#anrSettingPage .resultTip .successColor,
#anrSettingPage .resultTip .errorColor,
#anrSettingPage .resultTip .newEventPic,
#anrSettingPage .resultTip .confirmPic,
#anrSettingPage .resultTip .refusePic,
#anrSettingPage .resultTip .timeoutPic{
	margin-right:5px;
}
#anrSettingPage .newEventPic:before{
	color:#4D84FF;
	font-size:18px;
}
#anrSettingPage .confirmPic:before{
	color:#67D972;
	font-size:18px;
}
#anrSettingPage .refusePic:before{
	font-size:18px;
}
#anrSettingPage .timeoutPic:before{
	color:#F2B354;
	font-size:18px;
}
#anrSettingPage .el-icon-circle-close:before{
		color:#E88282;
		content:'\e6fb';
	}
</style>

<!--ANR 页面 -->
<div class="panelDefault commonWarp" id="anrSettingPage">
	<!-- 右上角操作按钮 -->
	<div v-show="activeName != 'pci' && optBtnShow" class="circleIcon placeholder-bt settingsBtn" placeholder="<%=rb.getString("SheZhi")%>" @click="settingBtn">
		<span class="el-icon el-icon-circle-setting"></span>
	</div>

	<!-- 表格 -->
	<el-tabs v-model='activeName' class="h100 newTabs" @tab-click='tabClick'>
		<el-tab-pane name='arNr' label="NR-NR ANR">
			<el-ctable
				ref='anrTable' :time="6"
				:id="'anr_list'" 
				:url='anrUrl' 
				row-key='id' 
				:query-params="anrParams" 
				height='100%' 
				page-size="50" 
				pagination="true">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input @keyup.enter.native="anrQuery" v-model="anrSearchText" class='pairgrid-query' 
							placeholder='<%=rb.getString("FuWuXiaoQuID")%>/<%=rb.getString("FuWuXiaoQuSSB")%>/<%=rb.getString("LinQuXiaoQuID")%>/<%=rb.getString("LTELinXiaoQuPinDian")%>'></el-input>
						<i class="el-icon-common-search el-icon" @click='anrQuery' style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column label='' width="30">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("FuWuXiaoQuID")%>' min-width="200" prop="serveCellId" show-overflow-tooltip="true"></el-table-column>				
				<el-table-column label='<%=rb.getString("FuWuXiaoQuPCI")%>' min-width="120" prop="scPCI" show-overflow-tooltip="true"></el-table-column>				
				<el-table-column label='<%=rb.getString("FuWuXiaoQuSSB")%>' min-width="200" prop="scFreq" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LinQuXiaoQuID")%>' min-width="150" prop="neighborCellID" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LTELinQuPCI")%>' min-width="120" prop="ncPCI" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LTELinXiaoQuPinDian")%>' min-width="120" prop="ncFreq" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LinQuGenZongQuYuMa")%>' min-width="180" prop="tac" show-overflow-tooltip="true"></el-table-column>				
				<el-table-column label='<%=rb.getString("TianJiaTuJing")%>' min-width="220" prop="origin" show-overflow-tooltip="true">
					<template slot-scope="scope">								
						<div v-if="scope.row.origin == '1'">							
							<span><%=rb.getString("TianJia")%></span>
						</div>
						<div v-if="scope.row.origin == '0'">															
							<span><%=rb.getString("ShanChu")%></span>
						</div>						
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("TianJiaShiJian")%>' min-width="160" prop="addTime" show-overflow-tooltip="true"></el-table-column>
				<el-table-column v-if="false" label='<%=rb.getString("DengDaiShiJian")%>' width="140" prop="anrTimeout" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="160" prop="status" show-overflow-tooltip="true">
					<template slot-scope="scope">								
						<div v-if="scope.row.status == 0" class="resultTip">									
							<span class="el-icon el-icon-status-newEvent newEventPic"></span>
							<span><%=rb.getString("XinShiJian")%></span>
						</div>
						<div v-if="scope.row.status == 1" class="resultTip">								
							<span class="el-icon el-icon-status-confirm confirmPic"></span>
							<span><%=rb.getString("YiQueRen")%></span>
						</div>
						<div v-if="scope.row.status == 2" class="resultTip">								
							<span class="el-icon el-icon-status-refuse refusePic"></span>
							<span><%=rb.getString("YiJuJue")%></span>
						</div>
						<div v-if="scope.row.status == 3" class="resultTip">									
							<span class="el-icon el-icon-status-timeOut timeoutPic"></span>
							<span><%=rb.getString("YiChaoShi")%></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' min-width="130" prop="result" show-overflow-tooltip="true">
					<template slot-scope="scope">								
						<div v-if="scope.row.result == '1'" class="resultTip">									
							<span class="el-icon el-icon-circle-success successColor"></span>
							<span><%=rb.getString("ChengGong")%></span>
						</div>
						<div v-else class="resultTip">									
							<span class="el-icon el-icon-circle-close errorColor"></span>
							<span><%=rb.getString("ShiBai")%></span>
						</div>						
					</template>
				</el-table-column>				
			</el-ctable>
			<el-cmenu ref="anrMenu" :data="menus" @click="clickMenu"></el-cmenu>
						
		</el-tab-pane>
		<!-- LTE邻区 -->
		<el-tab-pane name='nrLte' label="NR-LTE ANR">
			<el-ctable 
				ref='lteTable' :time="6"
				:id="'lte_list'" 
				:url='lteUrl' 
				row-key='id'  
				:query-params="lteParams" 
				height='100%' 
				page-size="50" 
				pagination="true">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input @keyup.enter.native="lteQuery" v-model="lteSearchText" class='pairgrid-query' placeholder='<%=rb.getString("FuWuXiaoQuID")%>/<%=rb.getString("FuWuXiaoQuSSB")%>/<%=rb.getString("LinQuXiaoQuID")%>/<%=rb.getString("LTELinXiaoQuPinDian")%>'></el-input>
						<i class="el-icon-common-search el-icon" @click='lteQuery' style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column label='' width="30">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>				
				<el-table-column label='<%=rb.getString("FuWuXiaoQuID")%>' min-width="200" prop="serveCellId" show-overflow-tooltip="true"></el-table-column>				
				<el-table-column label='<%=rb.getString("FuWuXiaoQuPCI")%>' min-width="120" prop="scPCI" show-overflow-tooltip="true"></el-table-column>				
				<el-table-column label='<%=rb.getString("FuWuXiaoQuSSB")%>' min-width="200" prop="scFreq" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LinQuGenZongQuYuMa")%>' min-width="150" prop=tac show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LTELinQuXiaoQuBiaoShi")%>' min-width="150" prop="ncID" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LTELinQuPCI")%>' min-width="120" prop="ncPCI" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("LTELinXiaoQuPinDian")%>' min-width="120" prop="ncFreq" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("TianJiaTuJing")%>' min-width="220" prop="origin" show-overflow-tooltip="true">				
					<template slot-scope="scope">								
						<div v-if="scope.row.origin == '1'">							
							<span><%=rb.getString("TianJia")%></span>
						</div>
						<div v-if="scope.row.origin == '0'">															
							<span><%=rb.getString("ShanChu")%></span>
						</div>						
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("TianJiaShiJian")%>' min-width="160" prop="addTime" show-overflow-tooltip="true"></el-table-column>
				<el-table-column v-if="false" label='<%=rb.getString("DengDaiShiJian")%>' width="180" prop="anrTimeout" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="160" prop="status" show-overflow-tooltip="true">
					<template slot-scope="scope">								
						<div v-if="scope.row.status == 0" class="resultTip">									
							<span class="el-icon el-icon-status-newEvent newEventPic"></span>
							<span><%=rb.getString("XinShiJian")%></span>
						</div>
						<div v-if="scope.row.status == 1" class="resultTip">								
							<span class="el-icon el-icon-status-confirm confirmPic"></span>
							<span><%=rb.getString("YiQueRen")%></span>
						</div>
						<div v-if="scope.row.status == 2" class="resultTip">								
							<span class="el-icon el-icon-status-refuse refusePic"></span>
							<span><%=rb.getString("YiJuJue")%></span>
						</div>
						<div v-if="scope.row.status == 3" class="resultTip">									
							<span class="el-icon el-icon-status-timeOut timeoutPic"></span>
							<span><%=rb.getString("YiChaoShi")%></span>
						</div>
					</template>
				</el-table-column>		
				<el-table-column label='<%=rb.getString("JieGuo")%>' min-width="110" prop="result" show-overflow-tooltip="true">
					<template slot-scope="scope">								
						<div v-if="scope.row.result == '1'" class="resultTip">									
							<span class="el-icon el-icon-circle-success successColor"></span>
							<span><%=rb.getString("ChengGong")%></span>
						</div>
						<div v-else class="resultTip">									
							<span class="el-icon el-icon-circle-close errorColor"></span>
							<span><%=rb.getString("ShiBai")%></span>
						</div>						
					</template>
				</el-table-column>				
			</el-ctable>
			<el-cmenu ref="lteMenu" :data="menus" @click="clickMenu"></el-cmenu>
		</el-tab-pane>
		<el-tab-pane name='pci' label="<%=rb.getString("PCIChongTuHunXiao")%>">
			<div id="pciBox" style="height:100%;"></div>
		</el-tab-pane>
	</el-tabs>
	
	<!-- slide -->
	<el-slide ref="slide" 
		:url="slideUrl" 
		:title="slideTitle" 
		:footer="slideFooter" 
		:position="slidePosition"
		:height="slideHeight" 
		:modal='modal'  
		:width="slideWidth" 
		@ok='saveSlide' 
		@cancel='cancelSlide' 
		:ok-text="'<%=rb.getString("QueDing")%>'" 
		:cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!--过滤冲突 slide -->
	<el-slide  ref="PCIConfusedSlide" :url='pciSlideUrl' :title="pciSlideTitle" :footer="pciSlideFooter" :header="pciSlideHeader" :position="pciSlidePosition" id="PCIConfusedSlide"
		:height="pciSlideHeight" :modal='modal' :width='pciSlideWidth' @ok="pciSlideOk"  @cancel="pciSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
</div>

<script type="text/javascript">
var anrSettingVue = new Vue({
	el:'#anrSettingPage',
	data(){
		var vm = this;

		return {
			activeName:'arNr',
			height:'100%',
			pageSize:50,
			//anr Table
			anrUrl:'${ctx}/anr/getANRList.action',
			anrSearchText:'',
			anrParams:{
				timeZone:timeZone,
				search_text:'',
				anrType:'nr',
				like_fields: 'serveCellId,scFreq,neighborCellID,ncFreq'
			},
			//lte Table
			lteUrl:'',	
			lteSearchText:'',
			lteParams:{
				timeZone:timeZone,
				search_text:'',
				anrType:'lte',
				like_fields: 'serveCellId,scFreq,neighborCellID,ncFreq'
			},						 
			menus:[],
			rowData:[],
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:false,
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
			modal:false,
			pciSlideUrl:'',
		    pciSlideTitle:'',
		    pciSlideHeader:'',
		    pciSlideFooter:'',
		    pciSlidePosition:'',
		    pciSlideHeight:'',
		    pciSlideWidth:''
		}
	},
	computed: {
		optBtnShow() {
			return writableMap['CODE_ADVANCE_SON'] == true;
		},
	},
	methods:{		
		//设置按钮
		settingBtn(){
			var vm = this;

	    	vm.slideUrl = "${ctx}/anr/goANRSetting.action";
	    	vm.slideTitle = '<%=rb.getString("SheZhi")%>';
    		vm.slideFooter = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
	    	vm.curType = 'setting';
	    	vm.$refs.slide.showSlide(function(){
	    		//eventBus.$emit('setting-config',vm.activeName)
	    	});
		},
		
		tabClick(){
			var vm = this;
			vm.$nextTick(function(){
				vm.lteUrl = '${ctx}/anr/getANRList.action';
				document.body.click();
			});
			if(vm.activeName == "pci"){
				$('#pciBox').addClass('loading');
				$("#pciBox").load('${ctx}/pci/goConflictConfusion.action',function(data){				
					$.parser.parse(this);
					$('#pciBox').removeClass('loading');
				});
			}
			//清空搜索条件
			vm.anrParams.search_text = '';
			vm.anrSearchText = '';
			vm.lteParams.search_text = '';
			vm.lteSearchText = '';
		},
		
		 //模糊查询
		anrQuery(){
			this.anrParams.search_text = this.anrSearchText;
		},
		
		 //模糊查询
		lteQuery(){
			this.lteParams.search_text = this.lteSearchText;
		},
		
		/**
		 * 操作栏点击展开下拉菜单
		 * @param row:点击的数据
		*/
	    optClick(row,ev){
		    var vm = this,
		        activeName = vm.$root.activeName;

	    	vm.rowData = row;
	    	vm.menus= [
	    		{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'}
		    ]

	    	vm.$nextTick(function(){
	    		document.body.click();

				if(activeName == 'arNr'){	    		
					vm.$refs.anrMenu.show(ev);
		    	}else if(activeName == 'nrLte'){
		    		vm.$refs.lteMenu.show(ev);
		    	}
	    	});
	    },
		/**
		 * 点击菜单
		 * @param ev:点击具体数据项进行筛选
		*/
	    clickMenu(ev){
	    	var codes = {
	    		view : this.viewConfig
	    	}
	    	if(codes[ev.code]){
	    		codes[ev.code](this.rowData.anrId)
	    	}
	    },
	    
	    //打开详情弹窗
	    viewConfig(id){
	    	var vm = this,
                activeName = vm.$root.activeName;

	    	vm.slideUrl = "${ctx}/anr/goANRInfo.action";
	    	vm.slideTitle = '<%=rb.getString("XinXi")%>';
    		vm.slideFooter = false;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
	    	vm.curType = 'view';
	    	vm.$refs.slide.showSlide(function(){
	    		eventBus.$emit('view-config',id,activeName);
	    	});
	    },
	    
	  	//点击页面其他地方菜单收起
		handerClose(){	  
        	if(this.activeName == 'arNr'){	    		
        		this.$refs.anrMenu.hide();
	    	}else{
	    		this.$refs.lteMenu.hide();
	    	}
	    },
	    
	 	// slide 保存
	    saveSlide(){
	    	eventBus.$emit('taskSave-ok');
	    },
		//关闭 slide 弹层
	    cancelSlide(){
	    	if(this.curType == 'view'){
	    		this.$refs.slide.hide();
	    	}else{
	    		eventBus.$emit('hander-cancel');
	    	}
	    },
		//保存后 关闭slide 弹层
	    closePage(){
			var vm = this;
	    	vm.$refs.slide.hide();
	    	vm.$refs.anrTable.refresh();
	    	vm.$refs.lteTable.refresh();
	    },

		pciSlideOk(){
			var vm = this;

			eventBus.$emit('confusedSetting-ok');
		},
		pciSlideCancel(){
			var vm = this;
			if(vm.slideType == 'set'){
				eventBus.$emit('confusedSetting-cancel');
			}else{
				vm.$refs.PCIConfusedSlide.hide();
			}
		},

	},
	
	mounted(){	
		eventBus.$off("cancel-settingPage").$on('cancel-settingPage',this.closePage);
	}
})
</script>